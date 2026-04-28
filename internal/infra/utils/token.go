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
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"google.golang.org/grpc/metadata"
)

const (
	TokenPrefix      = "mh_"
	RobotTokenPrefix = "robot_"
	tokenRandBytes   = 24 // 192 bits of entropy → 32 base64url chars

	// MaxTokenRetries caps the retry loop in the (astronomically unlikely) event of
	// a collision. In practice the loop never executes more than once.
	MaxTokenRetries = 3

	saltLength = 16
	letters    = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

// GenerateToken produces a raw token, its SHA-256 hash, and its prefix string.
//
// Format: mh_<base64url(24 random bytes)>
func GenerateToken(prefix string) (raw, hash string, err error) {
	buf := make([]byte, tokenRandBytes)
	if _, err = rand.Read(buf); err != nil {
		err = fmt.Errorf("failed to read random: %s", err)
		return
	}
	body := base64.RawURLEncoding.EncodeToString(buf)

	raw = prefix + body
	hash = Sha256Hex(raw)
	return
}

func GenerateRobotToken() (raw, hash string, err error) {
	return GenerateToken(RobotTokenPrefix)
}

func GenerateUserToken() (raw, hash string, err error) {
	return GenerateToken(TokenPrefix)
}

func Sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", sum)
}

func GenerateSalt() (string, error) {
	salt := make([]byte, saltLength)
	for i := range salt {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		salt[i] = letters[num.Int64()]
	}
	return string(salt), nil
}

// ParseBearerToken extracts the token from the request's Authorization header.
// Expects the format: "Authorization: Bearer <token>".
// Returns (token, true) on success, ("", false) if not present or malformed.
func ParseBearerToken(r *http.Request) (token string, ok bool) {
	auth := r.Header.Get("Authorization")
	bearer, found := strings.CutPrefix(auth, "Bearer ")
	if !found {
		return "", false
	}
	return strings.TrimSpace(bearer), true
}

func ParseBearerTokenFromGRPCContext(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.New("missing metadata")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", errors.New("missing authorization header")
	}
	bearer, found := strings.CutPrefix(values[0], "Bearer ")
	if !found {
		return "", errors.New("invalid authorization format, expected Bearer token")
	}
	return strings.TrimSpace(bearer), nil
}

// ParseBasicAuthFromGRPCContext extracts Basic Auth credentials from gRPC metadata.
// gRPC Gateway converts HTTP Authorization header to metadata with key "authorization".
// Returns (username, password, nil) on success.
func ParseBasicAuthFromGRPCContext(ctx context.Context) (username, password string, err error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", "", errors.New("missing metadata")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", "", errors.New("missing authorization header")
	}

	encoded, found := strings.CutPrefix(values[0], "Basic ")
	if !found {
		return "", "", errors.New("not a basic auth header")
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", "", errors.New("invalid base64 encoding")
	}

	username, password, found = strings.Cut(string(decoded), ":")
	if !found {
		return "", "", errors.New("invalid basic auth format")
	}

	return username, password, nil
}
