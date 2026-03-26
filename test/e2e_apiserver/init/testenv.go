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

package testenv

import (
	"log"
	"sync"
	"sync/atomic"

	"github.com/matrixhub-ai/matrixhub/test/tools"
)

var (
	once        sync.Once
	initialized int32
	refCount    int32
)

// InitTestEnvironment initializes the test environment
func InitTestEnvironment() {
	newCount := atomic.AddInt32(&refCount, 1)
	log.Printf("=== Reference count increased to: %d ===\n", newCount)

	once.Do(func() {
		log.Println("=== Initializing test environment ===")

		// Initialize auth (admin login) first
		err := tools.InitAuth()
		if err != nil {
			panic(err)
		}

		// Initialize HTTP clients (which depend on auth)
		err = tools.InitHTTPClients()
		if err != nil {
			panic(err)
		}

		atomic.StoreInt32(&initialized, 1)
		log.Println("=== Test environment initialized ===")
	})
}

// CleanupTestEnvironment cleans up the test environment
func CleanupTestEnvironment() {
	if atomic.LoadInt32(&initialized) == 0 {
		return
	}

	newCount := atomic.AddInt32(&refCount, -1)
	log.Printf("=== Reference count decreased to: %d ===\n", newCount)

	if newCount <= 0 {
		log.Println("=== Cleaning up test environment ===")
		tools.Close()
		atomic.StoreInt32(&initialized, 0)
		log.Println("=== Test environment cleaned up ===")
	}
}
