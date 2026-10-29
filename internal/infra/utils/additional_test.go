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

package utils_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
	"google.golang.org/grpc/metadata"

	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

func TestPaginationUtilities(t *testing.T) {
	require.True(t, utils.IsFullPageData(1, -1))
	require.False(t, utils.IsFullPageData(2, -1))
	require.Zero(t, utils.CalculatePages(0, 10))
	require.Zero(t, utils.CalculatePages(10, 0))
	require.Equal(t, int32(3), utils.CalculatePages(21, 10))

	page := utils.NewPage(0, 0)
	require.Equal(t, int32(1), page.Page)
	require.Equal(t, int32(10), page.PageSize)
	require.Same(t, page, utils.SetPageTotal(page, 21))
	require.Equal(t, int32(3), page.Pages)

	all := utils.NewPage(1, -1)
	utils.SetPageTotal(all, 21)
	require.Equal(t, int32(1), all.Pages)
	utils.SetPageTotal(all, 0)
	require.Zero(t, all.Pages)
}

func TestDurationCronAndRepositoryUtilities(t *testing.T) {
	require.Equal(t, "2d 3h 4m", utils.FormatDuration(51*time.Hour+4*time.Minute))
	require.Equal(t, "", utils.FormatDuration(30*time.Second))
	require.NoError(t, utils.ValidateCronExpr("*/5 * * * *"))
	require.Error(t, utils.ValidateCronExpr("invalid"))
	after := time.Date(2026, time.October, 29, 8, 1, 0, 0, time.UTC)
	next, err := utils.NextAfter("0 9 * * *", after)
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, time.October, 29, 9, 0, 0, 0, time.UTC), next)
	_, err = utils.NextAfter("invalid", after)
	require.Error(t, err)

	tests := []struct {
		input, repoType, project, name string
		ok                             bool
	}{
		{input: "org/model.git", repoType: "models", project: "org", name: "model", ok: true},
		{input: "datasets/org/data.git", repoType: "datasets", project: "org", name: "data", ok: true},
		{input: "spaces/org/app", repoType: "spaces", project: "org", name: "app", ok: true},
		{input: "invalid", repoType: "models", project: "invalid"},
	}
	for _, tt := range tests {
		repoType, project, name, ok := utils.ParseFromRepoName(tt.input)
		require.Equal(t, tt.repoType, repoType)
		require.Equal(t, tt.project, project)
		require.Equal(t, tt.name, name)
		require.Equal(t, tt.ok, ok)
	}
}

func TestTokenGenerationAndHTTPBearerParsing(t *testing.T) {
	userToken, userHash, err := utils.GenerateUserToken()
	require.NoError(t, err)
	require.Contains(t, userToken, utils.TokenPrefix)
	require.Equal(t, utils.Sha256Hex(userToken), userHash)

	robotToken, robotHash, err := utils.GenerateRobotToken()
	require.NoError(t, err)
	require.Contains(t, robotToken, utils.RobotTokenPrefix)
	require.Equal(t, utils.Sha256Hex(robotToken), robotHash)
	require.NotEqual(t, userToken, robotToken)

	salt, err := utils.GenerateSalt()
	require.NoError(t, err)
	require.Len(t, salt, 16)

	request, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)
	_, ok := utils.ParseBearerToken(request)
	require.False(t, ok)
	request.Header.Set("Authorization", "Basic abc")
	_, ok = utils.ParseBearerToken(request)
	require.False(t, ok)
	request.Header.Set("Authorization", "Bearer  token-value ")
	token, ok := utils.ParseBearerToken(request)
	require.True(t, ok)
	require.Equal(t, "token-value", token)
}

func TestGRPCAuthenticationParsing(t *testing.T) {
	_, err := utils.ParseBearerTokenFromGRPCContext(context.Background())
	require.ErrorContains(t, err, "missing metadata")

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer token-value"))
	token, err := utils.ParseBearerTokenFromGRPCContext(ctx)
	require.NoError(t, err)
	require.Equal(t, "token-value", token)

	ctx = metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Basic abc"))
	_, err = utils.ParseBearerTokenFromGRPCContext(ctx)
	require.ErrorContains(t, err, "expected Bearer")

	encoded := base64.StdEncoding.EncodeToString([]byte("alice:secret"))
	ctx = metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Basic "+encoded))
	username, password, err := utils.ParseBasicAuthFromGRPCContext(ctx)
	require.NoError(t, err)
	require.Equal(t, "alice", username)
	require.Equal(t, "secret", password)

	for _, header := range []string{"Bearer token", "Basic !!!", "Basic " + base64.StdEncoding.EncodeToString([]byte("missing-colon"))} {
		ctx = metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", header))
		_, _, err = utils.ParseBasicAuthFromGRPCContext(ctx)
		require.Error(t, err)
	}
	_, _, err = utils.ParseBasicAuthFromGRPCContext(context.Background())
	require.ErrorContains(t, err, "missing metadata")
}

func TestCookieFromGRPCContext(t *testing.T) {
	require.Empty(t, utils.GetCookieFromContext(context.Background()))
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("cookie", user.CookieName+"=session-token; other=value"))
	require.Equal(t, "session-token", utils.GetCookieFromContext(ctx))
	ctx = metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"grpcgateway-cookie", user.CookieName+"=preferred-token",
		"cookie", user.CookieName+"=fallback-token",
	))
	require.Equal(t, "preferred-token", utils.GetCookieFromContext(ctx))
}

func TestParseFingerprint(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	sshKey, err := ssh.NewPublicKey(publicKey)
	require.NoError(t, err)
	authorizedKey := string(ssh.MarshalAuthorizedKey(sshKey))

	fingerprint, err := utils.ParseFingerprint(authorizedKey)
	require.NoError(t, err)
	require.Equal(t, ssh.FingerprintSHA256(sshKey), fingerprint)
	require.Equal(t, fingerprint, requireFingerprint(t, fingerprint))
	_, err = utils.ParseFingerprint("SHA256:not base64!")
	require.Error(t, err)
	_, err = utils.ParseFingerprint("not-an-ssh-key")
	require.Error(t, err)
}

func requireFingerprint(t *testing.T, fingerprint string) string {
	t.Helper()
	parsed, err := utils.ParseFingerprint(fingerprint)
	require.NoError(t, err)
	return parsed
}
