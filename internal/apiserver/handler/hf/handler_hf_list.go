package hf

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/matrixhub-ai/hfd/pkg/hf"
	"github.com/matrixhub-ai/hfd/pkg/repository"
)

// handleList is the unified handler for listing repositories of different types (models, datasets, spaces).
func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoType := vars["repoType"]
	h.handleListRepos(w, r, repoType)
}

// repoListFilter holds the parsed query parameters for listing repositories.
type repoListFilter struct {
	search     string
	author     string
	filterTags []string
	sortField  string
	limit      int
	offset     int
}

// parseRepoListFilter extracts and parses the list query parameters from the request.
func parseRepoListFilter(r *http.Request) repoListFilter {
	query := r.URL.Query()
	f := repoListFilter{
		search:     query.Get("search"),
		author:     query.Get("author"),
		filterTags: query["filter"],
		sortField:  query.Get("sort"),
	}
	if v, err := strconv.Atoi(query.Get("limit")); err == nil && v > 0 {
		f.limit = v
	}
	if cursor := query.Get("cursor"); cursor != "" {
		f.offset = decodeCursorOffset(cursor)
	}
	return f
}

// handleListRepos is the unified handler for listing models, datasets, or spaces.
func (h *Handler) handleListRepos(w http.ResponseWriter, r *http.Request, repoType string) {
	f := parseRepoListFilter(r)

	reposDir := h.storage.RepositoriesDir()
	isModel := repoType == "models"

	baseDir := reposDir
	if !isModel {
		baseDir = filepath.Join(reposDir, repoType)
	}

	entries := discoverRepos(baseDir, isModel, f.author)
	items := buildRepoListItems(entries, isModel, f.search, f.filterTags)

	// Sort results
	sortRepoItems(items, f.sortField)

	// Apply cursor offset
	if f.offset > 0 && f.offset < len(items) {
		items = items[f.offset:]
	} else if f.offset >= len(items) {
		items = nil
	}

	// Apply limit and set Link header if there are more results
	if f.limit > 0 && len(items) > f.limit {
		items = items[:f.limit]

		nextCursor := encodeCursorOffset(f.offset + f.limit)
		nextURL := buildNextURL(r, nextCursor)
		w.Header().Set("Link", fmt.Sprintf("<%s>; rel=\"next\"", nextURL))
	}

	if items == nil {
		items = []repoListItem{}
	}

	responseJSON(w, items, http.StatusOK)
}

// repoEntry represents a discovered repository on disk.
type repoEntry struct {
	fullName string // "namespace/repo"
	repoPath string // absolute path to the .git directory
}

// discoverRepos walks the base directory and returns all valid repository entries,
// applying namespace-level filters (author, skipping non-model prefixes).
func discoverRepos(baseDir string, isModel bool, author string) []repoEntry {
	if author != "" {
		return discoverReposInNamespace(filepath.Join(baseDir, author), author)
	}
	namespaces, err := os.ReadDir(baseDir)
	if err != nil {
		return nil
	}

	var entries []repoEntry
	for _, nsEntry := range namespaces {
		if !nsEntry.IsDir() {
			continue
		}
		nsName := nsEntry.Name()

		// For models, skip the datasets/ and spaces/ directories
		if isModel && (nsName == "datasets" || nsName == "spaces") {
			continue
		}
		entries = append(entries, discoverReposInNamespace(filepath.Join(baseDir, nsName), nsName)...)
	}
	return entries
}

// discoverReposInNamespace returns all valid repository entries within a single namespace directory.
func discoverReposInNamespace(nsPath, nsName string) []repoEntry {
	repos, err := os.ReadDir(nsPath)
	if err != nil {
		return nil
	}

	var entries []repoEntry
	for _, repoDir := range repos {
		if !repoDir.IsDir() || !strings.HasSuffix(repoDir.Name(), ".git") {
			continue
		}
		repoPath := filepath.Join(nsPath, repoDir.Name())
		if !repository.IsRepository(repoPath) {
			continue
		}
		name := strings.TrimSuffix(repoDir.Name(), ".git")
		entries = append(entries, repoEntry{
			fullName: nsName + "/" + name,
			repoPath: repoPath,
		})
	}
	return entries
}

// buildRepoListItems converts discovered repo entries into list items,
// applying search and tag filters and reading metadata from each repository.
func buildRepoListItems(entries []repoEntry, isModel bool, search string, filterTags []string) []repoListItem {
	var items []repoListItem
	for _, e := range entries {
		if search != "" && !strings.Contains(strings.ToLower(e.fullName), strings.ToLower(search)) {
			continue
		}

		item := repoListItem{
			RepoID: e.fullName,
		}
		if isModel {
			item.ModelID = e.fullName
		}

		if repo, err := repository.Open(e.repoPath); err == nil {
			meta := collectRepoMetadata(repo, repo.DefaultBranch())
			item.Tags = meta.tags
			item.PipelineTag = meta.pipelineTag
			item.LibraryName = meta.libraryName
		}

		if len(filterTags) > 0 && !matchesAllTags(item.Tags, filterTags) {
			continue
		}

		items = append(items, item)
	}
	return items
}

// repoMetadata holds metadata extracted from a repository's README.md
// front matter and config.json. It is the single source of truth for
// tag collection, pipeline_tag, library_name, cardData, and createdAt,
// used by both the list endpoints and the individual repo info endpoint.
type repoMetadata struct {
	tags        []string
	pipelineTag string
	libraryName string
	cardData    any
}

// collectRepoMetadata reads metadata from an already-opened repository at the
// given revision. It extracts tags, pipeline_tag, library_name, and cardData
// from README.md YAML front matter and config.json, and derives createdAt
// from the latest commit date.
func collectRepoMetadata(repo *repository.Repository, rev string) repoMetadata {
	var meta repoMetadata

	seen := make(map[string]struct{})
	addTag := func(tag string) {
		if tag == "" {
			return
		}
		if _, ok := seen[tag]; !ok {
			seen[tag] = struct{}{}
			meta.tags = append(meta.tags, tag)
		}
	}

	// README.md YAML front matter
	if blob, err := repo.Blob(rev, "README.md"); err == nil {
		if rc, err := blob.NewReader(); err == nil {
			if rm, err := hf.ParseReadme(rc); err == nil {
				for _, tag := range rm.Tags() {
					addTag(tag)
				}
				meta.pipelineTag = rm.Card.PipelineTag
				meta.libraryName = rm.Card.LibraryName
				meta.cardData = rm.CardData
			}
			_ = rc.Close()
		}
	}

	// config.json
	if blob, err := repo.Blob(rev, "config.json"); err == nil {
		if rc, err := blob.NewReader(); err == nil {
			if cfg, err := hf.ParseConfigData(rc); err == nil {
				for _, tag := range cfg.Tags() {
					addTag(tag)
				}
			}
			_ = rc.Close()
		}
	}

	return meta
}

// matchesAllTags checks if the repo tags contain all the filter tags.
func matchesAllTags(repoTags, filterTags []string) bool {
	tagSet := make(map[string]struct{}, len(repoTags))
	for _, t := range repoTags {
		tagSet[t] = struct{}{}
	}
	for _, f := range filterTags {
		if _, ok := tagSet[f]; !ok {
			return false
		}
	}
	return true
}

// sortRepoItems sorts the list by the given field.
func sortRepoItems(items []repoListItem, sortField string) {
	switch sortField {
	case "downloads":
		sort.Slice(items, func(i, j int) bool {
			return items[i].Downloads > items[j].Downloads
		})
	case "likes":
		sort.Slice(items, func(i, j int) bool {
			return items[i].Likes > items[j].Likes
		})
	case "trending_score", "trendingScore":
		sort.Slice(items, func(i, j int) bool {
			return items[i].TrendingScore > items[j].TrendingScore
		})
	default:
		// Default: sort alphabetically by ID
		sort.Slice(items, func(i, j int) bool {
			return items[i].RepoID < items[j].RepoID
		})
	}
}

// cursorPayload is the JSON structure encoded inside the base64 cursor.
type cursorPayload struct {
	Offset int `json:"offset"`
}

// encodeCursorOffset encodes an offset into a base64 cursor string.
func encodeCursorOffset(offset int) string {
	data, _ := json.Marshal(cursorPayload{Offset: offset})
	return base64.URLEncoding.EncodeToString(data)
}

// decodeCursorOffset decodes a base64 cursor string into an offset.
// Returns 0 if the cursor is invalid.
func decodeCursorOffset(cursor string) int {
	data, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		// Try standard encoding as fallback
		data, err = base64.StdEncoding.DecodeString(cursor)
		if err != nil {
			return 0
		}
	}
	var payload cursorPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return 0
	}
	if payload.Offset < 0 {
		return 0
	}
	return payload.Offset
}

// buildNextURL constructs the full URL for the next page by replacing the
// cursor parameter while preserving all other query parameters.
func buildNextURL(r *http.Request, cursor string) string {
	origin := requestOrigin(r)
	q := make(url.Values)
	for k, vs := range r.URL.Query() {
		if k == "cursor" {
			continue
		}
		q[k] = vs
	}
	q.Set("cursor", cursor)
	return origin + r.URL.Path + "?" + q.Encode()
}
