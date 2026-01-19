package repository

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

// AgentsContent represents the content of an AGENTS.md file
// AGENTS.md follows the specification from https://agents.md - a simple,
// open format for guiding AI coding agents with plain markdown instructions.
type AgentsContent struct {
	Content string `json:"content"` // The full markdown content
}

const (
	// MaxAgentsFileSize limits AGENTS.md file size to prevent DoS
	MaxAgentsFileSize = 1024 * 1024 // 1MB
)

var (
	// ErrAgentsFileNotFound indicates AGENTS.md doesn't exist in repository
	ErrAgentsFileNotFound = errors.New("AGENTS.md not found")
	// ErrAgentsFileTooLarge indicates AGENTS.md exceeds size limit
	ErrAgentsFileTooLarge = errors.New("AGENTS.md file too large")
)

// GetAgentsContent retrieves the AGENTS.md file from the repository at the given ref.
// AGENTS.md is a plain markdown file containing instructions for AI coding agents.
// Returns ErrAgentsFileNotFound if the file doesn't exist, which is not an error condition.
func (r *Repository) GetAgentsContent(ref string) (*AgentsContent, error) {
	// Try to get the AGENTS.md blob from the repository root
	blob, err := r.Blob(ref, "AGENTS.md")
	if err != nil {
		// Check if it's a "not found" error
		if strings.Contains(err.Error(), "not found") {
			return nil, ErrAgentsFileNotFound
		}
		return nil, fmt.Errorf("failed to read AGENTS.md: %w", err)
	}

	// Check file size
	if blob.Size() > MaxAgentsFileSize {
		return nil, ErrAgentsFileTooLarge
	}

	// Read the file content
	reader, err := blob.NewReader()
	if err != nil {
		return nil, fmt.Errorf("failed to open AGENTS.md reader: %w", err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read AGENTS.md content: %w", err)
	}

	return &AgentsContent{
		Content: string(content),
	}, nil
}

// HasAgentsFile checks if the repository has an AGENTS.md file at the given ref
func (r *Repository) HasAgentsFile(ref string) bool {
	_, err := r.GetAgentsContent(ref)
	return err == nil
}
