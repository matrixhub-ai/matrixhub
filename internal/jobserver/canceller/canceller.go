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

package canceller

import (
	"context"
	"sync"
)

// Canceller manages cancel functions for in-flight jobs.
type Canceller interface {
	Register(jobID int, cancel context.CancelFunc)
	Unregister(jobID int)
	Cancel(jobID int) bool
}

type memCanceller struct {
	m sync.Map // jobID int -> context.CancelFunc
}

// NewMemCanceller returns an in-process Canceller.
func NewMemCanceller() Canceller {
	return &memCanceller{}
}

func (c *memCanceller) Register(jobID int, cancel context.CancelFunc) {
	c.m.Store(jobID, cancel)
}

func (c *memCanceller) Unregister(jobID int) {
	c.m.Delete(jobID)
}

func (c *memCanceller) Cancel(jobID int) bool {
	v, ok := c.m.Load(jobID)
	if !ok {
		return false
	}
	cancel, ok := v.(context.CancelFunc)
	if !ok {
		return false
	}
	cancel()
	return true
}
