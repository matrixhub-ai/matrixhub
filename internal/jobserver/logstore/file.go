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

package logstore

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FileLogStore writes job logs to the local filesystem.
type FileLogStore struct {
	baseDir string
}

// NewFileLogStore creates a FileLogStore, ensuring baseDir exists.
func NewFileLogStore(baseDir string) *FileLogStore {
	_ = os.MkdirAll(baseDir, 0755)
	return &FileLogStore{baseDir: baseDir}
}

func (f *FileLogStore) path(jobID int) string {
	return filepath.Join(f.baseDir, fmt.Sprintf("%d.log", jobID))
}

// Writer opens (or creates) the log file for appending.
func (f *FileLogStore) Writer(jobID int) (io.WriteCloser, error) {
	return os.OpenFile(f.path(jobID), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
}

// Reader opens the log file for reading.
func (f *FileLogStore) Reader(jobID int) (io.ReadCloser, error) {
	return os.Open(f.path(jobID))
}
