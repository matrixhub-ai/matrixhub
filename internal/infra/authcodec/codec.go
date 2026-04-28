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

package authcodec

import (
	"encoding/json"
	"fmt"

	"github.com/matrixhub-ai/matrixhub/internal/domain/auth"
	"github.com/matrixhub-ai/matrixhub/internal/domain/robot"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
)

type typedIdentity struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func Marshal(id auth.Identity) (string, error) {
	payload, err := json.Marshal(id)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	env := typedIdentity{
		Type:    id.TypeName(),
		Payload: payload,
	}

	b, err := json.Marshal(env)
	if err != nil {
		return "", fmt.Errorf("marshal envelope: %w", err)
	}

	return string(b), nil
}

func Unmarshal(s string) (auth.Identity, error) {
	var tid typedIdentity
	if err := json.Unmarshal([]byte(s), &tid); err != nil {
		return nil, fmt.Errorf("unmarshal typedIdentity: %w", err)
	}

	switch tid.Type {
	case "robot":
		var r robot.Identity
		if err := json.Unmarshal(tid.Payload, &r); err != nil {
			return nil, fmt.Errorf("unmarshal robot: %w", err)
		}
		return &r, nil

	case "user":
		var u user.Identity
		if err := json.Unmarshal(tid.Payload, &u); err != nil {
			return nil, fmt.Errorf("unmarshal user: %w", err)
		}
		return &u, nil

	default:
		return nil, fmt.Errorf("unknown identity type: %q", tid.Type)
	}
}
