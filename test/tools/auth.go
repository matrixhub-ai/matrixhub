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

package tools

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/antihax/optional"
	v1alpha1login "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/login"
	v1alpha1user "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/user"
)

var (
	authInitOnce sync.Once
	authInitErr  error

	// Admin credentials (from environment or default)
	adminUsername string
	adminPassword string
	adminCookie   string

	// API clients for admin
	adminLoginApi *v1alpha1login.LoginApiService
	adminUsersApi *v1alpha1user.UsersApiService
)

// Environment variable names for admin credentials
const (
	EnvAdminUsername = "MATRIXHUB_ADMIN_USERNAME"
	EnvAdminPassword = "MATRIXHUB_ADMIN_PASSWORD"
)

// Default admin credentials
const (
	DefaultAdminUsername = "admin"
	DefaultAdminPassword = "changeme"
)

// InitAuth initializes authentication with admin login
func InitAuth() error {
	authInitOnce.Do(func() {
		// Get admin credentials from environment or use defaults
		adminUsername = os.Getenv(EnvAdminUsername)
		if adminUsername == "" {
			adminUsername = DefaultAdminUsername
		}
		adminPassword = os.Getenv(EnvAdminPassword)
		if adminPassword == "" {
			adminPassword = DefaultAdminPassword
		}

		log.Printf("Initializing auth with admin user: %s\n", adminUsername)

		// Login as admin to get cookie
		cookie, err := LoginUser(adminUsername, adminPassword)
		if err != nil {
			authInitErr = fmt.Errorf("failed to login as admin: %w", err)
			return
		}

		adminCookie = cookie
		log.Println("Admin login successful")

		// Initialize admin API clients with cookie
		initAdminClients()
	})

	return authInitErr
}

// initAdminClients initializes API clients with admin cookie
func initAdminClients() {
	baseURL := GetBaseURL()

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402
			Proxy:           http.ProxyFromEnvironment,
		},
	}

	defaultHeaders := map[string]string{
		"Cookie":       adminCookie,
		"Content-Type": "application/json",
	}

	// Initialize Login API client
	loginCfg := &v1alpha1login.Configuration{
		BasePath:      baseURL,
		DefaultHeader: defaultHeaders,
		HTTPClient:    httpClient,
	}
	adminLoginApi = v1alpha1login.NewAPIClient(loginCfg).LoginApi

	// Initialize Users API client
	userCfg := &v1alpha1user.Configuration{
		BasePath:      baseURL,
		DefaultHeader: defaultHeaders,
		HTTPClient:    httpClient,
	}
	adminUsersApi = v1alpha1user.NewAPIClient(userCfg).UsersApi
}

// LoginUser logs in a user and returns the session cookie
func LoginUser(username, password string) (string, error) {
	baseURL := GetBaseURL()

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402
			Proxy:           http.ProxyFromEnvironment,
		},
	}

	cfg := &v1alpha1login.Configuration{
		BasePath: baseURL,
		DefaultHeader: map[string]string{
			"Content-Type": "application/json",
		},
		HTTPClient: httpClient,
	}

	loginApi := v1alpha1login.NewAPIClient(cfg).LoginApi

	ctx := context.Background()
	resp, httpResponse, err := loginApi.LoginLogin(ctx, v1alpha1login.V1alpha1LoginRequest{
		Username: username,
		Password: password,
	})
	if err != nil {
		return "", fmt.Errorf("login failed: %w", err)
	}

	// Extract cookie from response header
	if httpResponse != nil {
		// Try to get Set-Cookie header
		setCookie := httpResponse.Header.Get("Set-Cookie")
		if setCookie != "" {
			// Extract only the name=value part (before the first semicolon)
			cookie := extractCookieValue(setCookie)
			if cookie != "" {
				log.Printf("User %s logged in successfully\n", username)
				return cookie, nil
			}
		}

		// Fallback to the cookie field in response body
		if resp.Cookie != "" {
			cookie := extractCookieValue(resp.Cookie)
			if cookie != "" {
				log.Printf("User %s logged in successfully\n", username)
				return cookie, nil
			}
		}
	}
	return "", fmt.Errorf("no cookie received from login")
}

// GetAdminCookie returns the admin session cookie
func GetAdminCookie() string {
	if adminCookie == "" {
		err := InitAuth()
		if err != nil {
			panic(err)
		}
	}
	return adminCookie
}

// GetAdminLoginApi returns the Login API client with admin auth
func GetAdminLoginApi() *v1alpha1login.LoginApiService {
	if adminLoginApi == nil {
		err := InitAuth()
		if err != nil {
			panic(err)
		}
	}
	return adminLoginApi
}

// GetAdminUsersApi returns the Users API client with admin auth
func GetAdminUsersApi() *v1alpha1user.UsersApiService {
	if adminUsersApi == nil {
		err := InitAuth()
		if err != nil {
			panic(err)
		}
	}
	return adminUsersApi
}

// CreateUser creates a new user with admin privileges
func CreateUser(username, password string, isAdmin bool) error {
	ctx := context.Background()
	_, _, err := GetAdminUsersApi().UsersCreateUser(ctx, v1alpha1user.V1alpha1CreateUserRequest{
		Username: username,
		Password: password,
		IsAdmin:  isAdmin,
	})
	if err != nil {
		return fmt.Errorf("failed to create user %s: %w", username, err)
	}
	log.Printf("User %s created successfully (isAdmin: %v)\n", username, isAdmin)
	return nil
}

// CreateUserAndLogin creates a user and logs in, returning the cookie
func CreateUserAndLogin(username, password string, isAdmin bool) (string, error) {
	// Create user first
	err := CreateUser(username, password, isAdmin)
	if err != nil {
		return "", err
	}

	// Login to get cookie
	cookie, err := LoginUser(username, password)
	if err != nil {
		return "", fmt.Errorf("failed to login after creating user: %w", err)
	}

	return cookie, nil
}

// GetUserIDByUsername searches for a user by username and returns the numeric ID
func GetUserIDByUsername(username string) (int64, error) {
	ctx := context.Background()
	resp, _, err := GetAdminUsersApi().UsersListUsers(ctx, &v1alpha1user.UsersApiUsersListUsersOpts{
		Page:     optional.NewInt32(1),
		PageSize: optional.NewInt32(100),
		Search:   optional.NewString(username),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to list users: %w", err)
	}
	for _, u := range resp.Users {
		if u.Username == username {
			return u.Id, nil
		}
	}
	return 0, fmt.Errorf("user %s not found", username)
}

// CreateUserAndLoginWithID creates a user, logs in, and returns (userID, cookie, error)
func CreateUserAndLoginWithID(username, password string, isAdmin bool) (string, string, error) {
	cookie, err := CreateUserAndLogin(username, password, isAdmin)
	if err != nil {
		return "", "", err
	}

	userID, err := GetUserIDByUsername(username)
	if err != nil {
		return "", "", fmt.Errorf("failed to get user ID for %s: %w", username, err)
	}

	return fmt.Sprintf("%d", userID), cookie, nil
}

// DeleteUser deletes a user by ID using admin privileges
func DeleteUser(id int64) error {
	ctx := context.Background()
	_, _, err := GetAdminUsersApi().UsersDeleteUser(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete user %d: %w", id, err)
	}
	return nil
}

// RandomString generates a random string for test usernames
func RandomString() string {
	rand.New(rand.NewSource(time.Now().UnixNano())) // #nosec G404
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))] // #nosec G404
	}
	return string(b)
}

// GenerateTestUsername generates a unique username for testing
func GenerateTestUsername(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, RandomString())
}

// extractCookieValue extracts the cookie name=value part from a Set-Cookie header
// e.g., "session=abc123; Path=/; HttpOnly" -> "session=abc123"
func extractCookieValue(setCookie string) string {
	// Find the first semicolon which separates cookie from attributes
	idx := strings.Index(setCookie, ";")
	if idx > 0 {
		return strings.TrimSpace(setCookie[:idx])
	}
	// If no semicolon, return as-is (trimmed)
	return strings.TrimSpace(setCookie)
}
