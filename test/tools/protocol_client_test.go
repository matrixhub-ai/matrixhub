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

package tools

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSSHClientFixtureRepositoryURL(t *testing.T) {
	fixture := SSHClientFixture{
		Host: "127.0.0.1",
		Port: 30022,
	}

	require.Equal(t, "git@127.0.0.1:project/model.git", fixture.RepositoryURL("project", "model"))
}

func TestHFCLIXetEnvironment(t *testing.T) {
	t.Setenv("HF_HUB_DISABLE_XET", "1")
	t.Setenv("HF_XET_CACHE", "/inherited-cache")
	root := t.TempDir()
	environment := HFCLIXetEnvironment(root, "test-token")
	require.Equal(t, "0", environment["HF_HUB_DISABLE_XET"])
	require.Equal(t, filepath.Join(root, "hf", "xet"), environment["HF_XET_CACHE"])
	require.Equal(t, "test-token", environment["HF_TOKEN"])
	require.Equal(t, "1", HFCLIEnvironment(root, "test-token")["HF_HUB_DISABLE_XET"])
}
