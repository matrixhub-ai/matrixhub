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
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"sync"

	v1alpha1current_user "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/current_user"
	v1alpha1model "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/model"
	v1alpha1project "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/project"
	v1alpha1registry "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/registry"
	v1alpha1sync_policy "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/sync_policy"
	v1alpha1user "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/user"
)

var (
	httpInitOnce sync.Once
	httpInitErr  error

	// HTTP API clients (using admin cookie)
	v1alpha1ProjectsApi    *v1alpha1project.ProjectsApiService
	v1alpha1UsersApi       *v1alpha1user.UsersApiService
	v1alpha1CurrentUserApi *v1alpha1current_user.CurrentUserApiService
	v1alpha1ModelsApi      *v1alpha1model.ModelsApiService
	v1alpha1RegistriesApi  *v1alpha1registry.RegistriesApiService
	v1alpha1SyncPolicyApi  *v1alpha1sync_policy.SyncPolicyApiService
)

// InitHTTPClients initializes HTTP API clients with admin authentication
func InitHTTPClients() error {
	httpInitOnce.Do(func() {
		// First ensure auth is initialized (admin login)
		err := InitAuth()
		if err != nil {
			httpInitErr = fmt.Errorf("failed to initialize auth: %w", err)
			return
		}

		baseURL := GetBaseURL()
		cookie := GetAdminCookie()

		log.Printf("Initializing HTTP clients for MatrixHub at %s...\n", baseURL)

		// Common HTTP client configuration
		httpClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402
				Proxy:           http.ProxyFromEnvironment,
			},
		}

		defaultHeaders := map[string]string{
			"Cookie":       cookie,
			"Content-Type": "application/json",
		}

		// Initialize Project API client
		projectCfg := &v1alpha1project.Configuration{
			BasePath:      baseURL,
			DefaultHeader: defaultHeaders,
			HTTPClient:    httpClient,
		}
		v1alpha1ProjectsApi = v1alpha1project.NewAPIClient(projectCfg).ProjectsApi

		// Initialize User API client
		userCfg := &v1alpha1user.Configuration{
			BasePath:      baseURL,
			DefaultHeader: defaultHeaders,
			HTTPClient:    httpClient,
		}
		v1alpha1UsersApi = v1alpha1user.NewAPIClient(userCfg).UsersApi

		// Initialize CurrentUser API client
		currentUserCfg := &v1alpha1current_user.Configuration{
			BasePath:      baseURL,
			DefaultHeader: defaultHeaders,
			HTTPClient:    httpClient,
		}
		v1alpha1CurrentUserApi = v1alpha1current_user.NewAPIClient(currentUserCfg).CurrentUserApi

		// Initialize Model API client
		modelCfg := &v1alpha1model.Configuration{
			BasePath:      baseURL,
			DefaultHeader: defaultHeaders,
			HTTPClient:    httpClient,
		}
		v1alpha1ModelsApi = v1alpha1model.NewAPIClient(modelCfg).ModelsApi

		// Initialize Registry API client
		registryCfg := &v1alpha1registry.Configuration{
			BasePath:      baseURL,
			DefaultHeader: defaultHeaders,
			HTTPClient:    httpClient,
		}
		v1alpha1RegistriesApi = v1alpha1registry.NewAPIClient(registryCfg).RegistriesApi

		// Initialize SyncPolicy API client
		syncPolicyCfg := &v1alpha1sync_policy.Configuration{
			BasePath:      baseURL,
			DefaultHeader: defaultHeaders,
			HTTPClient:    httpClient,
		}
		v1alpha1SyncPolicyApi = v1alpha1sync_policy.NewAPIClient(syncPolicyCfg).SyncPolicyApi

		log.Println("HTTP clients initialized successfully")
	})

	return httpInitErr
}

// GetV1alpha1ProjectsApi returns the Projects HTTP API client
func GetV1alpha1ProjectsApi() *v1alpha1project.ProjectsApiService {
	if v1alpha1ProjectsApi == nil {
		err := InitHTTPClients()
		if err != nil {
			panic(err)
		}
	}
	return v1alpha1ProjectsApi
}

// GetV1alpha1UsersApi returns the Users HTTP API client
func GetV1alpha1UsersApi() *v1alpha1user.UsersApiService {
	if v1alpha1UsersApi == nil {
		err := InitHTTPClients()
		if err != nil {
			panic(err)
		}
	}
	return v1alpha1UsersApi
}

// GetV1alpha1CurrentUserApi returns the CurrentUser HTTP API client
func GetV1alpha1CurrentUserApi() *v1alpha1current_user.CurrentUserApiService {
	if v1alpha1CurrentUserApi == nil {
		err := InitHTTPClients()
		if err != nil {
			panic(err)
		}
	}
	return v1alpha1CurrentUserApi
}

// GetV1alpha1ModelsApi returns the Models HTTP API client
func GetV1alpha1ModelsApi() *v1alpha1model.ModelsApiService {
	if v1alpha1ModelsApi == nil {
		err := InitHTTPClients()
		if err != nil {
			panic(err)
		}
	}
	return v1alpha1ModelsApi
}

// GetV1alpha1RegistriesApi returns the Registries HTTP API client.
func GetV1alpha1RegistriesApi() *v1alpha1registry.RegistriesApiService {
	if v1alpha1RegistriesApi == nil {
		err := InitHTTPClients()
		if err != nil {
			panic(err)
		}
	}
	return v1alpha1RegistriesApi
}

// GetV1alpha1SyncPolicyApi returns the SyncPolicy HTTP API client.
func GetV1alpha1SyncPolicyApi() *v1alpha1sync_policy.SyncPolicyApiService {
	if v1alpha1SyncPolicyApi == nil {
		err := InitHTTPClients()
		if err != nil {
			panic(err)
		}
	}
	return v1alpha1SyncPolicyApi
}

// CreateModelClientWithCookie creates a new Model API client with a specific cookie
func CreateModelClientWithCookie(cookie string) *v1alpha1model.ModelsApiService {
	baseURL := GetBaseURL()

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402
			Proxy:           http.ProxyFromEnvironment,
		},
	}

	cfg := &v1alpha1model.Configuration{
		BasePath: baseURL,
		DefaultHeader: map[string]string{
			"Cookie":       cookie,
			"Content-Type": "application/json",
		},
		HTTPClient: httpClient,
	}

	return v1alpha1model.NewAPIClient(cfg).ModelsApi
}

// CreateAPIClientWithCookie creates a new API client with a specific cookie
// This is useful for testing with different users
func CreateProjectClientWithCookie(cookie string) *v1alpha1project.ProjectsApiService {
	baseURL := GetBaseURL()

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402
			Proxy:           http.ProxyFromEnvironment,
		},
	}

	cfg := &v1alpha1project.Configuration{
		BasePath: baseURL,
		DefaultHeader: map[string]string{
			"Cookie":       cookie,
			"Content-Type": "application/json",
		},
		HTTPClient: httpClient,
	}

	return v1alpha1project.NewAPIClient(cfg).ProjectsApi
}

// CreateCurrentUserClientWithCookie creates a new CurrentUser API client with a specific cookie
func CreateCurrentUserClientWithCookie(cookie string) *v1alpha1current_user.CurrentUserApiService {
	baseURL := GetBaseURL()

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402
			Proxy:           http.ProxyFromEnvironment,
		},
	}

	cfg := &v1alpha1current_user.Configuration{
		BasePath: baseURL,
		DefaultHeader: map[string]string{
			"Cookie":       cookie,
			"Content-Type": "application/json",
		},
		HTTPClient: httpClient,
	}

	return v1alpha1current_user.NewAPIClient(cfg).CurrentUserApi
}

// CreateUserClientWithCookie creates a new User API client with a specific cookie
func CreateUserClientWithCookie(cookie string) *v1alpha1user.UsersApiService {
	baseURL := GetBaseURL()

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402
			Proxy:           http.ProxyFromEnvironment,
		},
	}

	cfg := &v1alpha1user.Configuration{
		BasePath: baseURL,
		DefaultHeader: map[string]string{
			"Cookie":       cookie,
			"Content-Type": "application/json",
		},
		HTTPClient: httpClient,
	}

	return v1alpha1user.NewAPIClient(cfg).UsersApi
}
