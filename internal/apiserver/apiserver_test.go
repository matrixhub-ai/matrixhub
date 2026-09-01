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

package apiserver

import (
	"reflect"
	"testing"

	"github.com/matrixhub-ai/matrixhub/internal/infra/config"
)

func TestInitGitStorageDoesNotRegisterMetadataPostReceiveHook(t *testing.T) {
	server := &APIServer{
		config: &config.Config{DataDir: t.TempDir()},
	}
	server.initMirrorHooks()
	server.initGitStorage()

	postReceiveHook := reflect.ValueOf(server.gitStorage.sharedMirror).
		Elem().FieldByName("postReceiveHookFunc")
	if !postReceiveHook.IsNil() {
		t.Fatal("mirror must not register the metadata post-receive hook")
	}
}
