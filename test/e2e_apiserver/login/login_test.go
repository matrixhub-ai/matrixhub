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

package login_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"strings"

	v1alpha1login "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/login"
	"github.com/matrixhub-ai/matrixhub/test/tools"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// newLoginApi creates a fresh Login API client with no session cookie.
func newLoginApi() *v1alpha1login.LoginApiService {
	baseURL := tools.GetBaseURL()
	cfg := &v1alpha1login.Configuration{
		BasePath: baseURL,
		DefaultHeader: map[string]string{
			"Content-Type": "application/json",
		},
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402
				Proxy:           http.ProxyFromEnvironment,
			},
		},
	}
	return v1alpha1login.NewAPIClient(cfg).LoginApi
}

// newLoginApiWithCookie creates a Login API client pre-loaded with the given session cookie.
func newLoginApiWithCookie(cookie string) *v1alpha1login.LoginApiService {
	baseURL := tools.GetBaseURL()
	cfg := &v1alpha1login.Configuration{
		BasePath: baseURL,
		DefaultHeader: map[string]string{
			"Content-Type": "application/json",
			"Cookie":       cookie,
		},
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402
				Proxy:           http.ProxyFromEnvironment,
			},
		},
	}
	return v1alpha1login.NewAPIClient(cfg).LoginApi
}

// extractSetCookie parses the name=value part from a Set-Cookie header value.
func extractSetCookie(resp *http.Response) string {
	if resp == nil {
		return ""
	}
	raw := resp.Header.Get("Set-Cookie")
	if raw == "" {
		return ""
	}
	if idx := strings.Index(raw, ";"); idx > 0 {
		return strings.TrimSpace(raw[:idx])
	}
	return strings.TrimSpace(raw)
}

var _ = Describe("Login", Label("login"), func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	// ═══════════════════════════════════════════════════════════
	// 1. Login API
	// ═══════════════════════════════════════════════════════════
	Context("Login API", func() {
		It("should login successfully with valid admin credentials", Label("L00001"), func() {
			_, httpResp, err := newLoginApi().LoginLogin(ctx, v1alpha1login.V1alpha1LoginRequest{
				Username: tools.DefaultAdminUsername,
				Password: tools.DefaultAdminPassword,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(httpResp).NotTo(BeNil())
			Expect(httpResp.StatusCode).To(Equal(http.StatusOK))
		})

		It("should return a Set-Cookie header on successful login", Label("L00002"), func() {
			_, httpResp, err := newLoginApi().LoginLogin(ctx, v1alpha1login.V1alpha1LoginRequest{
				Username: tools.DefaultAdminUsername,
				Password: tools.DefaultAdminPassword,
			})
			Expect(err).NotTo(HaveOccurred())

			cookie := extractSetCookie(httpResp)
			Expect(cookie).NotTo(BeEmpty(), "Set-Cookie header must be present after login")
			GinkgoWriter.Printf("Session cookie: %s\n", cookie)
		})

		It("should grant authenticated access with the login cookie", Label("L00003"), func() {
			_, httpResp, err := newLoginApi().LoginLogin(ctx, v1alpha1login.V1alpha1LoginRequest{
				Username: tools.DefaultAdminUsername,
				Password: tools.DefaultAdminPassword,
			})
			Expect(err).NotTo(HaveOccurred())

			cookie := extractSetCookie(httpResp)
			Expect(cookie).NotTo(BeEmpty())

			currentUserApi := tools.CreateCurrentUserClientWithCookie(cookie)
			resp, _, err := currentUserApi.CurrentUserGetCurrentUser(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Username).To(Equal(tools.DefaultAdminUsername))

			GinkgoWriter.Printf("Authenticated as: %s (id=%d, isAdmin=%v)\n", resp.Username, resp.Id, resp.IsAdmin)
		})

		It("should fail to login with an incorrect password", Label("L00004"), func() {
			_, _, err := newLoginApi().LoginLogin(ctx, v1alpha1login.V1alpha1LoginRequest{
				Username: tools.DefaultAdminUsername,
				Password: "wrong-password-xyz",
			})
			Expect(err).To(HaveOccurred(), "incorrect password must be rejected")
			GinkgoWriter.Printf("Expected error: %v\n", err)
		})

		It("should fail to login with a non-existent username", Label("L00005"), func() {
			_, _, err := newLoginApi().LoginLogin(ctx, v1alpha1login.V1alpha1LoginRequest{
				Username: "no-such-user-xyz",
				Password: "somepassword",
			})
			Expect(err).To(HaveOccurred(), "non-existent user must be rejected")
			GinkgoWriter.Printf("Expected error: %v\n", err)
		})

		It("should fail to login with an empty username", Label("L00006"), func() {
			_, _, err := newLoginApi().LoginLogin(ctx, v1alpha1login.V1alpha1LoginRequest{
				Username: "",
				Password: tools.DefaultAdminPassword,
			})
			Expect(err).To(HaveOccurred(), "empty username must fail validation")
			GinkgoWriter.Printf("Expected error: %v\n", err)
		})

		It("should fail to login with an empty password", Label("L00007"), func() {
			_, _, err := newLoginApi().LoginLogin(ctx, v1alpha1login.V1alpha1LoginRequest{
				Username: tools.DefaultAdminUsername,
				Password: "",
			})
			Expect(err).To(HaveOccurred(), "empty password must fail validation")
			GinkgoWriter.Printf("Expected error: %v\n", err)
		})

		It("should login successfully with remember_me set to true", Label("L00008"), func() {
			_, httpResp, err := newLoginApi().LoginLogin(ctx, v1alpha1login.V1alpha1LoginRequest{
				Username:   tools.DefaultAdminUsername,
				Password:   tools.DefaultAdminPassword,
				RememberMe: true,
			})
			Expect(err).NotTo(HaveOccurred())

			cookie := extractSetCookie(httpResp)
			Expect(cookie).NotTo(BeEmpty(), "remember_me login must still issue a session cookie")
			GinkgoWriter.Printf("remember_me cookie: %s\n", cookie)
		})
	})

	// ═══════════════════════════════════════════════════════════
	// 2. Logout API
	// ═══════════════════════════════════════════════════════════
	Context("Logout API", func() {
		var (
			username string
			cookie   string
		)

		BeforeEach(func() {
			password := "Test@123456"
			username = tools.GenerateTestUsername("login-logout")
			var err error
			_, cookie, err = tools.CreateUserAndLoginWithID(username, password, false)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			userID, err := tools.GetUserIDByUsername(username)
			if err == nil {
				_ = tools.DeleteUser(userID)
			}
		})

		It("should logout successfully when authenticated", Label("L00009"), func() {
			_, _, err := newLoginApiWithCookie(cookie).LoginLogout(ctx, v1alpha1login.V1alpha1LogoutRequest{})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should invalidate the session cookie after logout", Label("L00010"), func() {
			_, _, err := newLoginApiWithCookie(cookie).LoginLogout(ctx, v1alpha1login.V1alpha1LogoutRequest{})
			Expect(err).NotTo(HaveOccurred())

			// The old cookie must no longer grant access.
			currentUserApi := tools.CreateCurrentUserClientWithCookie(cookie)
			_, _, err = currentUserApi.CurrentUserGetCurrentUser(ctx)
			Expect(err).To(HaveOccurred(), "stale cookie must be rejected after logout")
			GinkgoWriter.Printf("Expected auth error after logout: %v\n", err)
		})

		It("should fail to logout without a valid session", Label("L00011"), func() {
			_, _, err := newLoginApi().LoginLogout(ctx, v1alpha1login.V1alpha1LogoutRequest{})
			Expect(err).To(HaveOccurred(), "logout with no session must be rejected")
			GinkgoWriter.Printf("Expected error: %v\n", err)
		})
	})

	// ═══════════════════════════════════════════════════════════
	// 3. Multi-user login isolation
	// ═══════════════════════════════════════════════════════════
	Context("Multi-user session isolation", func() {
		It("should issue independent sessions for concurrent users", Label("L00012"), func() {
			password := "Test@123456"

			usernameA := tools.GenerateTestUsername("login-ua")
			usernameB := tools.GenerateTestUsername("login-ub")

			_, cookieA, err := tools.CreateUserAndLoginWithID(usernameA, password, false)
			Expect(err).NotTo(HaveOccurred())

			_, cookieB, err := tools.CreateUserAndLoginWithID(usernameB, password, false)
			Expect(err).NotTo(HaveOccurred())

			Expect(cookieA).NotTo(Equal(cookieB), "each user must receive a unique session cookie")

			apiA := tools.CreateCurrentUserClientWithCookie(cookieA)
			apiB := tools.CreateCurrentUserClientWithCookie(cookieB)

			respA, _, err := apiA.CurrentUserGetCurrentUser(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(respA.Username).To(Equal(usernameA))

			respB, _, err := apiB.CurrentUserGetCurrentUser(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(respB.Username).To(Equal(usernameB))

			GinkgoWriter.Printf("User A: %s (id=%d), User B: %s (id=%d)\n",
				respA.Username, respA.Id, respB.Username, respB.Id)

			// Cleanup
			idA, err := tools.GetUserIDByUsername(usernameA)
			if err == nil {
				_ = tools.DeleteUser(idA)
			}
			idB, err := tools.GetUserIDByUsername(usernameB)
			if err == nil {
				_ = tools.DeleteUser(idB)
			}
		})
	})
})
