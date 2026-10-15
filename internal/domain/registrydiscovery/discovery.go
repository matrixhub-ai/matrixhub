// Copyright The MatrixHub Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package registrydiscovery

import (
	"context"
	"errors"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
)

// Errors returned by RemoteFileLister implementations, wrapped with details.
var (
	// ErrRemoteNotFound means the remote repository or revision does not exist.
	ErrRemoteNotFound = errors.New("remote repository or revision not found")
	// ErrRemoteAccessDenied means the registry refused access, e.g. a gated
	// or private repository without a usable credential.
	ErrRemoteAccessDenied = errors.New("remote registry denied access")
	// ErrRemoteUnavailable means the registry could not be reached.
	ErrRemoteUnavailable = errors.New("remote registry unavailable")
)

// RemoteFileLister lists the files of a remote repository revision through a
// registry's Hugging Face-compatible API. It is read-only.
type RemoteFileLister interface {
	// ListRevisionFiles resolves revision (empty means the default branch) on
	// the registry and lists every file of that commit with LFS details.
	ListRevisionFiles(ctx context.Context, reg *registry.Registry, namespace, name, revision string) (*git.RevisionFiles, error)
}

// Discovery discovers remote repositories for a given registry type.
type Discovery interface {
	// ListRepositories returns remote repositories matching the filter.
	// The registry credential (if any) is used for authenticated access.
	ListRepositories(ctx context.Context, reg *registry.Registry, filter Filter) ([]RemoteRepository, error)
}

// Filter holds criteria for listing remote repositories.
type Filter struct {
	Namespace    string // e.g., "google"
	ResourceType string // "model", "dataset"
}

// RemoteRepository represents a repository discovered on a remote registry.
type RemoteRepository struct {
	Namespace    string
	Name         string
	ResourceType string
}

// Provider keys — used as stable map keys for Discovery registrations.
const (
	ProviderHuggingFace = "HUGGINGFACE"
	ProviderMatrixHub   = "MATRIXHUB"
)

// KeyFromRegistryType converts a stored registry.Type (proto enum string, e.g.
// "REGISTRY_TYPE_HUGGINGFACE") to a stable provider key. If the type is not
// recognised, the original value is returned unchanged.
func KeyFromRegistryType(t string) string {
	switch t {
	case "REGISTRY_TYPE_HUGGINGFACE":
		return ProviderHuggingFace
	case "REGISTRY_TYPE_MATRIXHUB":
		return ProviderMatrixHub
	default:
		return t
	}
}
