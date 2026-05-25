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

package current_user_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	v1alpha1current_user "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/current_user"
	"github.com/matrixhub-ai/matrixhub/test/tools"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// generateSSHPublicKey generates a fresh ed25519 key-pair on every call and
// returns the public key in OpenSSH authorized_keys format.  Each call
// produces a unique fingerprint, so tests never collide on the global
// uix_ssh_keys_fingerprint unique index.
func generateSSHPublicKey() string {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	pub, err := ssh.NewPublicKey(priv.Public())
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pub)))
}

var _ = Describe("CurrentUser", Label("current-user"), func() {
	var (
		ctx            context.Context
		currentUserApi *v1alpha1current_user.CurrentUserApiService
		username       string
		password       string
		userID         int32
	)

	BeforeEach(func() {
		ctx = context.Background()
		password = "Test@123456"
		username = tools.GenerateTestUsername("cu")
		var cookie string
		var err error
		userID, cookie, err = tools.CreateUserAndLoginWithID(username, password, false)
		Expect(err).NotTo(HaveOccurred())
		currentUserApi = tools.CreateCurrentUserClientWithCookie(cookie)
	})

	AfterEach(func() {
		_ = tools.DeleteUser(int64(userID))
	})

	// ═══════════════════════════════════════════════════════════
	// 1. GetCurrentUser API
	// ═══════════════════════════════════════════════════════════
	Context("GetCurrentUser API", func() {
		It("should return the correct identity for the logged-in user", Label("CU00001"), func() {
			resp, _, err := currentUserApi.CurrentUserGetCurrentUser(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Username).To(Equal(username))
			Expect(resp.Id).NotTo(BeZero())
			Expect(resp.IsAdmin).To(BeFalse())

			GinkgoWriter.Printf("CurrentUser: id=%d, username=%s, isAdmin=%v\n", resp.Id, resp.Username, resp.IsAdmin)
		})

		It("should reflect isAdmin=true for an admin user", Label("CU00002"), func() {
			adminCookie, err := tools.LoginUser(tools.DefaultAdminUsername, tools.DefaultAdminPassword)
			Expect(err).NotTo(HaveOccurred())

			adminApi := tools.CreateCurrentUserClientWithCookie(adminCookie)
			resp, _, err := adminApi.CurrentUserGetCurrentUser(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Username).To(Equal(tools.DefaultAdminUsername))
			Expect(resp.IsAdmin).To(BeTrue())
		})
	})

	// ═══════════════════════════════════════════════════════════
	// 2. Access Token API
	// ═══════════════════════════════════════════════════════════
	Context("Access Token API", func() {
		It("should return an empty list when no tokens exist", Label("CU00003"), func() {
			resp, _, err := currentUserApi.CurrentUserListAccessTokens(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Items).To(BeEmpty())
		})

		It("should create an access token and return a non-empty token string", Label("CU00004"), func() {
			resp, _, err := currentUserApi.CurrentUserCreateAccessToken(ctx, v1alpha1current_user.V1alpha1CreateAccessTokenRequest{
				Name: "e2e-token",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Token).NotTo(BeEmpty())

			GinkgoWriter.Printf("Created access token prefix: %s...\n", resp.Token[:12])
		})

		It("should fail to create an access token with an empty name", Label("CU00005"), func() {
			_, _, err := currentUserApi.CurrentUserCreateAccessToken(ctx, v1alpha1current_user.V1alpha1CreateAccessTokenRequest{
				Name: "",
			})
			Expect(err).To(HaveOccurred(), "empty name must fail validation")
		})

		It("should list access tokens including the newly created one", Label("CU00006"), func() {
			_, _, err := currentUserApi.CurrentUserCreateAccessToken(ctx, v1alpha1current_user.V1alpha1CreateAccessTokenRequest{
				Name: "e2e-list-token",
			})
			Expect(err).NotTo(HaveOccurred())

			listResp, _, err := currentUserApi.CurrentUserListAccessTokens(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(listResp.Items).To(HaveLen(1))
			Expect(listResp.Items[0].Name).To(Equal("e2e-list-token"))

			GinkgoWriter.Printf("Listed %d access token(s)\n", len(listResp.Items))
		})

		It("should delete an access token successfully", Label("CU00007"), func() {
			createResp, _, err := currentUserApi.CurrentUserCreateAccessToken(ctx, v1alpha1current_user.V1alpha1CreateAccessTokenRequest{
				Name: "e2e-delete-token",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(createResp.Token).NotTo(BeEmpty())

			listResp, _, err := currentUserApi.CurrentUserListAccessTokens(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(listResp.Items).To(HaveLen(1))
			tokenID := listResp.Items[0].Id

			_, _, err = currentUserApi.CurrentUserDeleteAccessToken(ctx, tokenID)
			Expect(err).NotTo(HaveOccurred())

			listAfter, _, err := currentUserApi.CurrentUserListAccessTokens(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(listAfter.Items).To(BeEmpty())
		})

		It("should create an access token with a future expiry date", Label("CU00008"), func() {
			expireAt := fmt.Sprintf("%d", time.Now().AddDate(1, 0, 0).Unix())
			resp, _, err := currentUserApi.CurrentUserCreateAccessToken(ctx, v1alpha1current_user.V1alpha1CreateAccessTokenRequest{
				Name:     "e2e-expiry-token",
				ExpireAt: expireAt,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Token).NotTo(BeEmpty())

			listResp, _, err := currentUserApi.CurrentUserListAccessTokens(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(listResp.Items).To(HaveLen(1))
			Expect(listResp.Items[0].ExpiredAt).NotTo(BeEmpty())
		})
	})

	// ═══════════════════════════════════════════════════════════
	// 3. SSH Key API
	// ═══════════════════════════════════════════════════════════
	Context("SSH Key API", func() {
		It("should return an empty list when no SSH keys exist", Label("CU00009"), func() {
			resp, _, err := currentUserApi.CurrentUserListSSHKeys(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Items).To(BeEmpty())
		})

		It("should create an SSH key successfully", Label("CU00010"), func() {
			_, _, err := currentUserApi.CurrentUserCreateSSHKey(ctx, v1alpha1current_user.V1alpha1CreateSshKeyRequest{
				Name:      "e2e-ssh-key",
				PublicKey: generateSSHPublicKey(),
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail to create an SSH key with an empty name", Label("CU00011"), func() {
			_, _, err := currentUserApi.CurrentUserCreateSSHKey(ctx, v1alpha1current_user.V1alpha1CreateSshKeyRequest{
				Name:      "",
				PublicKey: generateSSHPublicKey(),
			})
			Expect(err).To(HaveOccurred(), "empty name must fail validation")
		})

		It("should fail to create an SSH key with an invalid public key", Label("CU00012"), func() {
			_, _, err := currentUserApi.CurrentUserCreateSSHKey(ctx, v1alpha1current_user.V1alpha1CreateSshKeyRequest{
				Name:      "invalid-key",
				PublicKey: "not-a-valid-public-key",
			})
			Expect(err).To(HaveOccurred(), "invalid public key format must be rejected")
		})

		It("should list SSH keys including the newly created one", Label("CU00013"), func() {
			_, _, err := currentUserApi.CurrentUserCreateSSHKey(ctx, v1alpha1current_user.V1alpha1CreateSshKeyRequest{
				Name:      "e2e-list-ssh-key",
				PublicKey: generateSSHPublicKey(),
			})
			Expect(err).NotTo(HaveOccurred())

			listResp, _, err := currentUserApi.CurrentUserListSSHKeys(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(listResp.Items).To(HaveLen(1))
			Expect(listResp.Items[0].Name).To(Equal("e2e-list-ssh-key"))

			GinkgoWriter.Printf("Listed %d SSH key(s)\n", len(listResp.Items))
		})

		It("should delete an SSH key successfully", Label("CU00014"), func() {
			_, _, err := currentUserApi.CurrentUserCreateSSHKey(ctx, v1alpha1current_user.V1alpha1CreateSshKeyRequest{
				Name:      "e2e-delete-ssh-key",
				PublicKey: generateSSHPublicKey(),
			})
			Expect(err).NotTo(HaveOccurred())

			listResp, _, err := currentUserApi.CurrentUserListSSHKeys(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(listResp.Items).To(HaveLen(1))
			keyID := listResp.Items[0].Id

			_, _, err = currentUserApi.CurrentUserDeleteSSHKey(ctx, keyID)
			Expect(err).NotTo(HaveOccurred())

			listAfter, _, err := currentUserApi.CurrentUserListSSHKeys(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(listAfter.Items).To(BeEmpty())
		})

		It("should reject a duplicate SSH public key for the same user", Label("CU00015"), func() {
			// Use the same key twice to exercise the duplicate-fingerprint guard.
			dupKey := generateSSHPublicKey()
			_, _, err := currentUserApi.CurrentUserCreateSSHKey(ctx, v1alpha1current_user.V1alpha1CreateSshKeyRequest{
				Name:      "e2e-dup-key-1",
				PublicKey: dupKey,
			})
			Expect(err).NotTo(HaveOccurred())

			_, _, err = currentUserApi.CurrentUserCreateSSHKey(ctx, v1alpha1current_user.V1alpha1CreateSshKeyRequest{
				Name:      "e2e-dup-key-2",
				PublicKey: dupKey,
			})
			Expect(err).To(HaveOccurred(), "duplicate SSH public key fingerprint must be rejected")
		})
	})

	// ═══════════════════════════════════════════════════════════
	// 4. ResetPassword API
	// ═══════════════════════════════════════════════════════════
	Context("ResetPassword API", func() {
		It("should reset password successfully with the correct old password", Label("CU00016"), func() {
			newPassword := "NewTest@654321"
			_, _, err := currentUserApi.CurrentUserResetPassword(ctx, v1alpha1current_user.V1alpha1ResetPasswordRequest{
				OldPassword: password,
				NewPassword: newPassword,
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify the new password works for login.
			_, err = tools.LoginUser(username, newPassword)
			Expect(err).NotTo(HaveOccurred())

			GinkgoWriter.Printf("Password reset successful for user: %s\n", username)
		})

		It("should fail to reset password with an incorrect old password", Label("CU00017"), func() {
			_, _, err := currentUserApi.CurrentUserResetPassword(ctx, v1alpha1current_user.V1alpha1ResetPasswordRequest{
				OldPassword: "wrong-old-password",
				NewPassword: "NewTest@654321",
			})
			Expect(err).To(HaveOccurred(), "wrong old password must be rejected")
		})

		It("should fail to reset password when old and new passwords are identical", Label("CU00018"), func() {
			_, _, err := currentUserApi.CurrentUserResetPassword(ctx, v1alpha1current_user.V1alpha1ResetPasswordRequest{
				OldPassword: password,
				NewPassword: password,
			})
			Expect(err).To(HaveOccurred(), "new password same as old must be rejected")
		})
	})

	// ═══════════════════════════════════════════════════════════
	// 5. GetProjectRoles API
	// ═══════════════════════════════════════════════════════════
	Context("GetProjectRoles API", func() {
		It("should return an empty map for a user with no project memberships", Label("CU00019"), func() {
			resp, _, err := currentUserApi.CurrentUserGetProjectRoles(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.ProjectRoles).To(BeEmpty())

			GinkgoWriter.Printf("Project roles: %v\n", resp.ProjectRoles)
		})
	})
})

