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

package ssh

import (
	"testing"

	"github.com/matrixhub-ai/hfd/pkg/repository"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name      string
		cmdLine   string
		service   string
		repoName  string
		operation string
		wantErr   bool
	}{
		{
			name:     "git-upload-pack with quotes",
			cmdLine:  "git-upload-pack '/repo.git'",
			service:  repository.GitUploadPack,
			repoName: "/repo.git",
		},
		{
			name:     "git-upload-pack without quotes",
			cmdLine:  "git-upload-pack /repo.git",
			service:  repository.GitUploadPack,
			repoName: "/repo.git",
		},
		{
			name:     "git-receive-pack with quotes",
			cmdLine:  "git-receive-pack '/repo.git'",
			service:  repository.GitReceivePack,
			repoName: "/repo.git",
		},
		{
			name:      "git-lfs-authenticate download",
			cmdLine:   "git-lfs-authenticate '/repo.git' download",
			service:   repository.GitLFSAuthenticate,
			repoName:  "/repo.git",
			operation: "download",
		},
		{
			name:      "git-lfs-authenticate upload",
			cmdLine:   "git-lfs-authenticate '/repo.git' upload",
			service:   repository.GitLFSAuthenticate,
			repoName:  "/repo.git",
			operation: "upload",
		},
		{
			name:      "git-lfs-authenticate without quotes",
			cmdLine:   "git-lfs-authenticate /user/model upload",
			service:   repository.GitLFSAuthenticate,
			repoName:  "/user/model",
			operation: "upload",
		},
		{
			name:      "git-lfs-transfer download",
			cmdLine:   "git-lfs-transfer '/repo.git' download",
			service:   repository.GitLFSTransfer,
			repoName:  "/repo.git",
			operation: "download",
		},
		{
			name:      "git-lfs-transfer upload",
			cmdLine:   "git-lfs-transfer '/repo.git' upload",
			service:   repository.GitLFSTransfer,
			repoName:  "/repo.git",
			operation: "upload",
		},
		{
			name:    "unsupported service",
			cmdLine: "git-foo '/repo.git'",
			wantErr: true,
		},
		{
			name:    "no arguments",
			cmdLine: "git-upload-pack",
			wantErr: true,
		},
		{
			name:    "empty string",
			cmdLine: "",
			wantErr: true,
		},
		{
			name:    "git-lfs-authenticate missing operation",
			cmdLine: "git-lfs-authenticate '/repo.git'",
			wantErr: true,
		},
		{
			name:    "git-lfs-transfer missing operation",
			cmdLine: "git-lfs-transfer '/repo.git'",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := parseCommand(tt.cmdLine)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cmd.service != tt.service {
				t.Errorf("service = %q, want %q", cmd.service, tt.service)
			}
			if cmd.repoName != tt.repoName {
				t.Errorf("repoName = %q, want %q", cmd.repoName, tt.repoName)
			}
			if cmd.operation != tt.operation {
				t.Errorf("operation = %q, want %q", cmd.operation, tt.operation)
			}
		})
	}
}
