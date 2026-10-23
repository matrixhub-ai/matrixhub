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

package utils

import "strings"

var repoTypePrefixes = []struct {
	prefix   string
	repoType string
}{
	{"datasets/", "datasets"},
	{"spaces/", "spaces"},
}

func ParseFromRepoName(repoName string) (repoType, project, name string, ok bool) {
	repoType = "models"
	namespacedName := repoName
	for _, p := range repoTypePrefixes {
		if strings.HasPrefix(repoName, p.prefix) {
			repoType = p.repoType
			namespacedName = strings.TrimPrefix(repoName, p.prefix)
			break
		}
	}

	project, name, ok = strings.Cut(namespacedName, "/")
	name = strings.TrimSuffix(name, ".git")
	return
}

// CanonicalRepoName makes HTTP and SSH repository names share one identity.
// Reject ambiguous components instead of cleaning traversal into another repo.
func CanonicalRepoName(name string) (string, string, bool) {
	repoType, project, model, ok := ParseFromRepoName(strings.TrimPrefix(name, "/"))
	for _, segment := range []string{project, model} {
		if segment == "" || segment == "." || segment == ".." || strings.ContainsAny(segment, "/\\\x00\r\n") {
			return "", "", false
		}
	}
	if !ok {
		return "", "", false
	}
	canonical := project + "/" + model
	if repoType != "models" {
		canonical = repoType + "/" + canonical
	}
	return canonical, repoType, true
}
