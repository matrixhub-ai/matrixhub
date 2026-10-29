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

package registry_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
)

type unsupportedCredential struct{}

func (unsupportedCredential) Type() string       { return "unsupported" }
func (unsupportedCredential) String() string     { return "value" }
func (unsupportedCredential) Value() interface{} { return nil }

func TestBasicCredentialRoundTrip(t *testing.T) {
	credential := registry.NewBasicCredential("alice", "secret")
	require.Equal(t, registry.CredentialTypeBasic, credential.Type())
	require.JSONEq(t, `{"username":"alice","password":"secret"}`, credential.String())
	require.Same(t, credential, credential.Value())
	require.Equal(t, "alice", registry.AsBasic(credential).Username)
	require.Nil(t, registry.AsBasic(unsupportedCredential{}))

	parsed := registry.ParseCredentialInfo(registry.CredentialTypeBasic, credential.String())
	require.Equal(t, registry.AsBasic(credential), registry.AsBasic(parsed))
	require.Nil(t, registry.ParseCredentialInfo("unknown", credential.String()))
	require.Nil(t, registry.ParseCredentialInfo(registry.CredentialTypeBasic, ""))
	require.Nil(t, registry.ParseCredentialInfo(registry.CredentialTypeBasic, `{`))
}

func TestRegistryCredentialPersistence(t *testing.T) {
	reg := &registry.Registry{}
	require.Equal(t, "registries", reg.TableName())
	require.Nil(t, reg.GetCredential())

	reg.SetCredential(registry.NewBasicCredential("robot", "token"))
	require.Equal(t, registry.CredentialTypeBasic, reg.CredentialType)
	require.Equal(t, "robot", registry.AsBasic(reg.GetCredential()).Username)

	reg.SetCredential(nil)
	require.Empty(t, reg.CredentialType)
	require.Empty(t, reg.AuthInfo)
}
