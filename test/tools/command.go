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
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// CommandResult captures a black-box CLI invocation without including its
// environment, which may contain access tokens.
type CommandResult struct {
	Name   string
	Args   []string
	Stdout string
	Stderr string
	Err    error
}

// FailureMessage returns diagnostics suitable for a Ginkgo assertion.
func (r CommandResult) FailureMessage() string {
	return fmt.Sprintf("command %q failed: %v\nstdout:\n%s\nstderr:\n%s",
		append([]string{r.Name}, r.Args...), r.Err, r.Stdout, r.Stderr)
}

// RunCommand executes a command directly, without a shell. Environment
// overrides are merged by key so host credentials cannot accidentally win
// over the isolated values supplied by a test.
func RunCommand(ctx context.Context, dir string, overrides map[string]string, name string, args ...string) CommandResult {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = commandEnvironment(overrides)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	return CommandResult{
		Name:   name,
		Args:   append([]string(nil), args...),
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		Err:    err,
	}
}

func commandEnvironment(overrides map[string]string) []string {
	env := make(map[string]string, len(os.Environ())+len(overrides))
	for _, item := range os.Environ() {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			env[key] = value
		}
	}
	for key, value := range overrides {
		env[key] = value
	}

	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, key+"="+env[key])
	}
	return result
}
