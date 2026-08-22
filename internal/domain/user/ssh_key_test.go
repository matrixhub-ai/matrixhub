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

package user

import (
	"testing"
	"time"
)

func TestSSHKeyIsExpired(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	past := now.Add(-time.Second)
	future := now.Add(time.Second)

	tests := []struct {
		name     string
		expireAt *time.Time
		want     bool
	}{
		{name: "never expires", expireAt: nil, want: false},
		{name: "past expiration", expireAt: &past, want: true},
		{name: "expiration instant", expireAt: &now, want: true},
		{name: "future expiration", expireAt: &future, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := SSHKey{ExpireAt: tt.expireAt}
			if got := key.IsExpired(now); got != tt.want {
				t.Fatalf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}
