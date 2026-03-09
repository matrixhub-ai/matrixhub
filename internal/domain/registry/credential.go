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

package registry

import "encoding/json"

const CredentialTypeBasic = "basic"

// ICredential is the domain interface for registry auth (basic, token, etc.).
type ICredential interface {
	Type() string
	String() string // persistence form (e.g. JSON)
	Value() interface{}
}

// BasicCredential is the domain value object for basic (username/password) auth.
type BasicCredential struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (c *BasicCredential) Type() string {
	return CredentialTypeBasic
}

func (c *BasicCredential) String() string {
	b, err := json.Marshal(c)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func (c *BasicCredential) Value() interface{} { return c }

func NewBasicCredential(username, password string) ICredential {
	return &BasicCredential{
		Username: username,
		Password: password,
	}
}

func AsBasic(c ICredential) *BasicCredential {
	if b, ok := c.(*BasicCredential); ok {
		return b
	}
	return nil
}

// ParseCredentialInfo builds an ICredential from stored CredentialType and AuthInfo; returns nil when empty or unknown.
func ParseCredentialInfo(credType, authInfo string) ICredential {
	if credType != CredentialTypeBasic || authInfo == "" {
		return nil
	}

	var b BasicCredential
	if err := json.Unmarshal([]byte(authInfo), &b); err != nil {
		return nil
	}
	return &b
}
