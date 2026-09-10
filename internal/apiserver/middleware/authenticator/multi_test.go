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

package authenticator

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/matrixhub-ai/matrixhub/internal/domain/auth"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
)

type stubAuthenticator struct {
	identity auth.Identity
	next, ok bool
	err      error
	calls    int
}

func (stub *stubAuthenticator) Authenticate(context.Context, *http.Request) (auth.Identity, bool, bool, error) {
	stub.calls++
	return stub.identity, stub.next, stub.ok, stub.err
}

func (stub *stubAuthenticator) AuthenticateToken(ctx context.Context, _, _ string) (auth.Identity, bool, bool, error) {
	return stub.Authenticate(ctx, nil)
}

func TestMultiAuthenticator(t *testing.T) {
	identity := user.NewUserIdentity(1, "alice")
	storageErr := errors.New("database unavailable")
	for _, test := range []struct {
		name     string
		stubs    []*stubAuthenticator
		calls    []int
		success  int
		next, ok bool
		err      error
	}{
		{"next", []*stubAuthenticator{{next: true}, {next: true}}, []int{1, 1}, -1, true, false, nil},
		{"rejected", []*stubAuthenticator{{next: true}, {}, {identity: identity, ok: true}}, []int{1, 1, 0}, -1, false, false, nil},
		{"success", []*stubAuthenticator{{next: true}, {identity: identity, ok: true}}, []int{1, 1}, 1, false, true, nil},
		{"error", []*stubAuthenticator{{err: storageErr}, {identity: identity, ok: true}}, []int{1, 0}, -1, false, false, storageErr},
	} {
		t.Run(test.name, func(t *testing.T) {
			authenticators := make([]HTTPAuthenticator, len(test.stubs))
			for index, stub := range test.stubs {
				authenticators[index] = stub
			}
			succeeded, gotIdentity, next, ok, err := NewMultiAuthenticator(authenticators...).AuthenticateToken(context.Background(), "alice", "token")
			if next != test.next || ok != test.ok || !errors.Is(err, test.err) {
				t.Fatalf("got next=%v ok=%v err=%v, want next=%v ok=%v err=%v", next, ok, err, test.next, test.ok, test.err)
			}
			if test.success >= 0 {
				if succeeded != test.stubs[test.success] || gotIdentity != identity {
					t.Errorf("unexpected successful authenticator or identity: %v, %v", succeeded, gotIdentity)
				}
			} else if succeeded != nil || gotIdentity != nil {
				t.Errorf("unexpected success: %v, %v", succeeded, gotIdentity)
			}
			for index, stub := range test.stubs {
				if stub.calls != test.calls[index] {
					t.Errorf("authenticator %d called %d times, want %d", index, stub.calls, test.calls[index])
				}
			}
		})
	}
}
