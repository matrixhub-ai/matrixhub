package repository

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// AgentMetadata represents the parsed metadata from an agent.md file
type AgentMetadata struct {
	Name        string   `json:"name" yaml:"name"`
	Version     string   `json:"version" yaml:"version"`
	Type        string   `json:"type" yaml:"type"`
	Tags        []string `json:"tags" yaml:"tags"`
	Description string   `json:"description" yaml:"description"`
	RawContent  string   `json:"rawContent,omitempty" yaml:"-"`
}

const (
	// MaxAgentFileSize limits agent.md file size to prevent DoS
	MaxAgentFileSize = 1024 * 1024 // 1MB
)

var (
	// ErrAgentFileNotFound indicates agent.md doesn't exist in repository
	ErrAgentFileNotFound = errors.New("agent.md not found")
	// ErrAgentFileTooLarge indicates agent.md exceeds size limit
	ErrAgentFileTooLarge = errors.New("agent.md file too large")
)

// GetAgentMetadata retrieves and parses the agent.md file from the repository at the given ref.
// Returns ErrAgentFileNotFound if the file doesn't exist, which is not an error condition.
func (r *Repository) GetAgentMetadata(ref string) (*AgentMetadata, error) {
	// Try to get the agent.md blob from the repository root
	blob, err := r.Blob(ref, "agent.md")
	if err != nil {
		// Check if it's a "not found" error
		if strings.Contains(err.Error(), "not found") {
			return nil, ErrAgentFileNotFound
		}
		return nil, fmt.Errorf("failed to read agent.md: %w", err)
	}

	// Check file size
	if blob.Size() > MaxAgentFileSize {
		return nil, ErrAgentFileTooLarge
	}

	// Read the file content
	reader, err := blob.NewReader()
	if err != nil {
		return nil, fmt.Errorf("failed to open agent.md reader: %w", err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent.md content: %w", err)
	}

	// Parse the content
	metadata, err := parseAgentMetadata(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse agent.md: %w", err)
	}

	// Store raw content
	metadata.RawContent = string(content)

	return metadata, nil
}

// parseAgentMetadata parses the agent.md content, extracting YAML frontmatter and markdown body
func parseAgentMetadata(content []byte) (*AgentMetadata, error) {
	metadata := &AgentMetadata{}

	// Check for YAML frontmatter
	scanner := bufio.NewScanner(bytes.NewReader(content))
	
	// Look for opening frontmatter delimiter
	if !scanner.Scan() {
		// Empty file
		return metadata, nil
	}

	firstLine := strings.TrimSpace(scanner.Text())
	
	// Check if file starts with YAML frontmatter (---)
	if firstLine == "---" {
		// Collect frontmatter content
		var frontmatter []string
		foundClosing := false
		
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) == "---" {
				foundClosing = true
				break
			}
			frontmatter = append(frontmatter, line)
		}

		if foundClosing && len(frontmatter) > 0 {
			// Parse YAML frontmatter
			yamlContent := strings.Join(frontmatter, "\n")
			if err := yaml.Unmarshal([]byte(yamlContent), metadata); err != nil {
				// Log but don't fail - frontmatter is optional
				// Return metadata with what we could parse
				return metadata, nil
			}
		}

		// Extract description from remaining markdown content
		var bodyLines []string
		for scanner.Scan() {
			line := scanner.Text()
			bodyLines = append(bodyLines, line)
		}
		
		// If no description in frontmatter, extract from body
		if metadata.Description == "" && len(bodyLines) > 0 {
			metadata.Description = extractDescription(bodyLines)
		}
	} else {
		// No frontmatter, treat entire content as description
		allContent := string(content)
		metadata.Description = extractDescription(strings.Split(allContent, "\n"))
	}

	return metadata, nil
}

// extractDescription extracts a meaningful description from markdown content
func extractDescription(lines []string) string {
	var descLines []string
	inCodeBlock := false
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Skip empty lines at the start
		if len(descLines) == 0 && trimmed == "" {
			continue
		}
		
		// Handle code blocks
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		
		// Skip headers (we want the content after headers)
		if strings.HasPrefix(trimmed, "#") && len(descLines) == 0 {
			continue
		}
		
		// Skip lines in code blocks
		if inCodeBlock {
			continue
		}
		
		// Add content lines
		if trimmed != "" {
			descLines = append(descLines, trimmed)
			// Limit description to first paragraph (about 3-5 lines)
			if len(descLines) >= 5 {
				break
			}
		} else if len(descLines) > 0 {
			// Empty line after content - end of first paragraph
			break
		}
	}
	
	return strings.Join(descLines, " ")
}

// HasAgentMetadata checks if the repository has an agent.md file at the given ref
func (r *Repository) HasAgentMetadata(ref string) bool {
	_, err := r.GetAgentMetadata(ref)
	return err == nil
}
