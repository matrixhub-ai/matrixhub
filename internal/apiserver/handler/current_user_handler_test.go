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

package handler

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
)

type sshKeyServiceStub struct {
	createErr error
}

func (s *sshKeyServiceStub) CreateSSHKey(context.Context, user.SSHKey) error {
	return s.createErr
}

func TestCurrentUserHandlerCreateSSHKeyAlreadyExists(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
	}{
		{name: "active key", repoErr: user.ErrSSHKeyAlreadyExists},
		{name: "expired key", repoErr: user.ErrSSHKeyExpired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &CurrentUserHandler{
				sshKeyService: &sshKeyServiceStub{createErr: tt.repoErr},
			}

			_, err := handler.CreateSSHKey(context.Background(), &v1alpha1.CreateSSHKeyRequest{
				Name:      "existing-key",
				PublicKey: "SHA256:AA",
			})
			if status.Code(err) != codes.AlreadyExists {
				t.Fatalf("status code = %s, want %s (err: %v)", status.Code(err), codes.AlreadyExists, err)
			}
			if status.Convert(err).Message() != tt.repoErr.Error() {
				t.Fatalf("status message = %q, want %q", status.Convert(err).Message(), tt.repoErr.Error())
			}
		})
	}
}
