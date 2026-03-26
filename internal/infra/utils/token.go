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

package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/pkg/errors"
)

const (
	tokenPrefix    = "mh_"
	tokenRandBytes = 24 // 192 bits of entropy → 32 base64url chars

	// MaxTokenRetries caps the retry loop in the (astronomically unlikely) event of
	// a collision. In practice the loop never executes more than once.
	MaxTokenRetries = 3
)

// GenerateToken produces a raw token, its SHA-256 hash, and its prefix string.
//
// Format: mh_<base64url(24 random bytes)>
func GenerateToken() (raw, hash string, err error) {
	buf := make([]byte, tokenRandBytes)
	if _, err = rand.Read(buf); err != nil {
		err = errors.Errorf("failed to read random: %s", err)
		return
	}

	// hf + "_" + base64url (no padding)
	body := base64.RawURLEncoding.EncodeToString(buf)

	raw = tokenPrefix + body
	hash = Sha256Hex(raw)
	return
}

func Sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", sum)
}
