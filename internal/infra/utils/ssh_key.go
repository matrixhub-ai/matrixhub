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
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"

	"golang.org/x/crypto/ssh"
)

// ParseFingerprint extracts a standard SHA256 fingerprint from either a raw
// public key string or an already-formatted "SHA256:..." fingerprint string.
func ParseFingerprint(credential string) (string, error) {
	credential = strings.TrimSpace(credential)

	if strings.HasPrefix(credential, "SHA256:") {
		b64 := strings.TrimPrefix(credential, "SHA256:")
		if _, err := base64.StdEncoding.DecodeString(b64); err != nil {
			return "", errors.New("invalid fingerprint encoding")
		}
		return credential, nil
	}

	pub, _, _, _, err := ssh.ParseAuthorizedKey([]byte(credential))
	if err != nil {
		return "", errors.New("invalid ssh public key format")
	}
	h := sha256.Sum256(pub.Marshal())
	return "SHA256:" + base64.StdEncoding.EncodeToString(h[:]), nil
}
