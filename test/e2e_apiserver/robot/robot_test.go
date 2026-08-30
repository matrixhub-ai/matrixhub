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

package robot_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"strings"

	"github.com/antihax/optional"
	v1alpha1robot "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/robot"
	v1alpha1user "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/user"
	"github.com/matrixhub-ai/matrixhub/test/tools"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// robotPrefix is prepended to every robot name by the server.
const robotPrefix = "robot$"

// generateRobotName returns a unique robot name (without prefix) for each test.
func generateRobotName(prefix string) string {
	return tools.GenerateTestUsername(prefix)
}

func robotStatus(status v1alpha1robot.V1alpha1RobotAccountStatus) *v1alpha1robot.V1alpha1RobotAccountStatus {
	return &status
}

func newRobotUsersAPI() *v1alpha1user.UsersApiService {
	cfg := &v1alpha1user.Configuration{
		BasePath: tools.GetBaseURL(),
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
	return v1alpha1user.NewAPIClient(cfg).UsersApi
}

func robotAuthContext(username, token string) context.Context {
	return context.WithValue(context.Background(), v1alpha1user.ContextBasicAuth, v1alpha1user.BasicAuth{
		UserName: username,
		Password: token,
	})
}

var _ = Describe("Robot", Label("robot"), func() {
	var (
		ctx       context.Context
		robotsApi *v1alpha1robot.RobotsApiService
	)

	BeforeEach(func() {
		ctx = context.Background()
		robotsApi = tools.GetV1alpha1RobotsApi()
	})

	// ═══════════════════════════════════════════════════════════
	// 1. CreateRobotAccount API
	// ═══════════════════════════════════════════════════════════
	Context("CreateRobotAccount API", func() {
		It("should create a robot account and return a non-empty token", Label("R00001", "smoke"), func() {
			name := generateRobotName("rb-create")
			resp, _, err := robotsApi.RobotsCreateRobotAccount(ctx, v1alpha1robot.V1alpha1CreateRobotAccountRequest{
				Name:        name,
				Description: "e2e test robot",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Token).NotTo(BeEmpty())

			// Cleanup: find the robot and delete it.
			listResp, _, err := robotsApi.RobotsListRobotAccounts(ctx, &v1alpha1robot.RobotsApiRobotsListRobotAccountsOpts{
				Search: optional.NewString(robotPrefix + name),
			})
			Expect(err).NotTo(HaveOccurred())
			for _, r := range listResp.Items {
				if r.Name == robotPrefix+name {
					_, _, _ = robotsApi.RobotsDeleteRobotAccount(ctx, r.Id)
				}
			}

			GinkgoWriter.Printf("Created robot token prefix: %s...\n", resp.Token[:12])
		})

		It("should fail to create a robot account with an empty name", Label("R00002"), func() {
			_, _, err := robotsApi.RobotsCreateRobotAccount(ctx, v1alpha1robot.V1alpha1CreateRobotAccountRequest{
				Name: "",
			})
			Expect(err).To(HaveOccurred(), "empty name must fail validation")
		})

		It("should fail to create a robot account with a duplicate name", Label("R00003"), func() {
			name := generateRobotName("rb-dup")
			_, _, err := robotsApi.RobotsCreateRobotAccount(ctx, v1alpha1robot.V1alpha1CreateRobotAccountRequest{
				Name: name,
			})
			Expect(err).NotTo(HaveOccurred())

			_, _, err = robotsApi.RobotsCreateRobotAccount(ctx, v1alpha1robot.V1alpha1CreateRobotAccountRequest{
				Name: name,
			})
			Expect(err).To(HaveOccurred(), "duplicate robot name must be rejected")

			// Cleanup
			listResp, _, _ := robotsApi.RobotsListRobotAccounts(ctx, &v1alpha1robot.RobotsApiRobotsListRobotAccountsOpts{
				Search: optional.NewString(robotPrefix + name),
			})
			for _, r := range listResp.Items {
				if r.Name == robotPrefix+name {
					_, _, _ = robotsApi.RobotsDeleteRobotAccount(ctx, r.Id)
				}
			}
		})

		It("should store the robot name with the 'robot$' prefix", Label("R00004"), func() {
			name := generateRobotName("rb-prefix")
			_, _, err := robotsApi.RobotsCreateRobotAccount(ctx, v1alpha1robot.V1alpha1CreateRobotAccountRequest{
				Name: name,
			})
			Expect(err).NotTo(HaveOccurred())

			listResp, _, err := robotsApi.RobotsListRobotAccounts(ctx, &v1alpha1robot.RobotsApiRobotsListRobotAccountsOpts{
				Search: optional.NewString(robotPrefix + name),
			})
			Expect(err).NotTo(HaveOccurred())

			found := false
			var foundID int64
			for _, r := range listResp.Items {
				if r.Name == robotPrefix+name {
					found = true
					foundID = r.Id
					Expect(strings.HasPrefix(r.Name, robotPrefix)).To(BeTrue())
					break
				}
			}
			Expect(found).To(BeTrue(), "robot must be listed with its prefixed name")

			_, _, _ = robotsApi.RobotsDeleteRobotAccount(ctx, foundID)
		})

		It("should create a robot with expiry and reflect VALID expire status", Label("R00005"), func() {
			name := generateRobotName("rb-expiry")
			_, _, err := robotsApi.RobotsCreateRobotAccount(ctx, v1alpha1robot.V1alpha1CreateRobotAccountRequest{
				Name:       name,
				ExpireDays: 30,
			})
			Expect(err).NotTo(HaveOccurred())

			listResp, _, err := robotsApi.RobotsListRobotAccounts(ctx, &v1alpha1robot.RobotsApiRobotsListRobotAccountsOpts{
				Search: optional.NewString(robotPrefix + name),
			})
			Expect(err).NotTo(HaveOccurred())

			for _, r := range listResp.Items {
				if r.Name == robotPrefix+name {
					Expect(r.ExpireDays).To(Equal(int32(30)))
					Expect(r.ExpireStatus).NotTo(BeNil())
					Expect(*r.ExpireStatus).To(Equal(v1alpha1robot.VALID_V1alpha1RobotAccountExpireStatus))

					_, _, _ = robotsApi.RobotsDeleteRobotAccount(ctx, r.Id)
					break
				}
			}
		})
	})

	// ═══════════════════════════════════════════════════════════
	// 2. GetRobotAccount / ListRobotAccounts APIs
	// ═══════════════════════════════════════════════════════════
	Context("GetRobotAccount and ListRobotAccounts APIs", func() {
		var (
			robotName string
			robotID   int64
		)

		BeforeEach(func() {
			robotName = generateRobotName("rb-get")
			_, _, err := robotsApi.RobotsCreateRobotAccount(ctx, v1alpha1robot.V1alpha1CreateRobotAccountRequest{
				Name:        robotName,
				Description: "get-test robot",
			})
			Expect(err).NotTo(HaveOccurred())

			listResp, _, err := robotsApi.RobotsListRobotAccounts(ctx, &v1alpha1robot.RobotsApiRobotsListRobotAccountsOpts{
				Search: optional.NewString(robotPrefix + robotName),
			})
			Expect(err).NotTo(HaveOccurred())
			for _, r := range listResp.Items {
				if r.Name == robotPrefix+robotName {
					robotID = r.Id
					break
				}
			}
			Expect(robotID).NotTo(BeZero())
		})

		AfterEach(func() {
			if robotID != 0 {
				_, _, _ = robotsApi.RobotsDeleteRobotAccount(ctx, robotID)
			}
		})

		It("should get a robot account by ID with all expected fields", Label("R00006"), func() {
			resp, _, err := robotsApi.RobotsGetRobotAccount(ctx, robotID)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Id).To(Equal(robotID))
			Expect(resp.Name).To(Equal(robotPrefix + robotName))
			Expect(resp.Description).To(Equal("get-test robot"))
			Expect(resp.CreatedAt).NotTo(BeEmpty())

			GinkgoWriter.Printf("GetRobot: id=%d, name=%s\n", resp.Id, resp.Name)
		})

		It("should fail to get a robot account with a non-existent ID", Label("R00007"), func() {
			_, _, err := robotsApi.RobotsGetRobotAccount(ctx, 999999999)
			Expect(err).To(HaveOccurred(), "non-existent robot ID must return an error")
		})

		It("should include the robot in the list", Label("R00008"), func() {
			resp, _, err := robotsApi.RobotsListRobotAccounts(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Pagination).NotTo(BeNil())
			Expect(resp.Pagination.Total).To(BeNumerically(">=", 1))

			found := false
			for _, r := range resp.Items {
				if r.Id == robotID {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "created robot must appear in list")
		})

		It("should support search in list", Label("R00009"), func() {
			resp, _, err := robotsApi.RobotsListRobotAccounts(ctx, &v1alpha1robot.RobotsApiRobotsListRobotAccountsOpts{
				Search: optional.NewString(robotPrefix + robotName),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Items).NotTo(BeEmpty())
			Expect(resp.Items[0].Name).To(Equal(robotPrefix + robotName))
		})
	})

	// ═══════════════════════════════════════════════════════════
	// 3. UpdateRobotAccount API
	// ═══════════════════════════════════════════════════════════
	Context("UpdateRobotAccount API", func() {
		var (
			robotName string
			robotID   int64
		)

		BeforeEach(func() {
			robotName = generateRobotName("rb-upd")
			_, _, err := robotsApi.RobotsCreateRobotAccount(ctx, v1alpha1robot.V1alpha1CreateRobotAccountRequest{
				Name:        robotName,
				Description: "original description",
			})
			Expect(err).NotTo(HaveOccurred())

			listResp, _, err := robotsApi.RobotsListRobotAccounts(ctx, &v1alpha1robot.RobotsApiRobotsListRobotAccountsOpts{
				Search: optional.NewString(robotPrefix + robotName),
			})
			Expect(err).NotTo(HaveOccurred())
			for _, r := range listResp.Items {
				if r.Name == robotPrefix+robotName {
					robotID = r.Id
					break
				}
			}
			Expect(robotID).NotTo(BeZero())
		})

		AfterEach(func() {
			if robotID != 0 {
				_, _, _ = robotsApi.RobotsDeleteRobotAccount(ctx, robotID)
			}
		})

		It("should update the robot description successfully", Label("R00010"), func() {
			_, _, err := robotsApi.RobotsUpdateRobotAccount(ctx, robotID, v1alpha1robot.RobotsUpdateRobotAccountBody{
				Description: "updated description",
				Status:      robotStatus(v1alpha1robot.ENABLED_V1alpha1RobotAccountStatus),
			})
			Expect(err).NotTo(HaveOccurred())

			resp, _, err := robotsApi.RobotsGetRobotAccount(ctx, robotID)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Description).To(Equal("updated description"))

			GinkgoWriter.Printf("Updated robot description: %s\n", resp.Description)
		})

		It("should update the robot to never-expire when expire_days is set to 0", Label("R00011"), func() {
			_, _, err := robotsApi.RobotsUpdateRobotAccount(ctx, robotID, v1alpha1robot.RobotsUpdateRobotAccountBody{
				ExpireDays: 0,
				Status:     robotStatus(v1alpha1robot.ENABLED_V1alpha1RobotAccountStatus),
			})
			Expect(err).NotTo(HaveOccurred())

			resp, _, err := robotsApi.RobotsGetRobotAccount(ctx, robotID)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.ExpireDays).To(Equal(int32(0)))
			Expect(*resp.ExpireStatus).To(Equal(v1alpha1robot.NEVER_V1alpha1RobotAccountExpireStatus))
		})

		It("should fail to update a non-existent robot account", Label("R00012"), func() {
			_, _, err := robotsApi.RobotsUpdateRobotAccount(ctx, 999999999, v1alpha1robot.RobotsUpdateRobotAccountBody{
				Description: "ghost",
			})
			Expect(err).To(HaveOccurred(), "updating non-existent robot must return an error")
		})
	})

	// ═══════════════════════════════════════════════════════════
	// 4. RefreshRobotAccountToken API
	// ═══════════════════════════════════════════════════════════
	Context("RefreshRobotAccountToken API", func() {
		var (
			robotName    string
			robotID      int64
			initialToken string
		)

		BeforeEach(func() {
			robotName = generateRobotName("rb-refresh")
			createResp, _, err := robotsApi.RobotsCreateRobotAccount(ctx, v1alpha1robot.V1alpha1CreateRobotAccountRequest{
				Name: robotName,
			})
			Expect(err).NotTo(HaveOccurred())
			initialToken = createResp.Token

			listResp, _, err := robotsApi.RobotsListRobotAccounts(ctx, &v1alpha1robot.RobotsApiRobotsListRobotAccountsOpts{
				Search: optional.NewString(robotPrefix + robotName),
			})
			Expect(err).NotTo(HaveOccurred())
			for _, r := range listResp.Items {
				if r.Name == robotPrefix+robotName {
					robotID = r.Id
					break
				}
			}
			Expect(robotID).NotTo(BeZero())
		})

		AfterEach(func() {
			if robotID != 0 {
				_, _, _ = robotsApi.RobotsDeleteRobotAccount(ctx, robotID)
			}
		})

		It("should auto-generate a new token different from the initial one", Label("R00013"), func() {
			resp, _, err := robotsApi.RobotsRefreshRobotAccountToken(ctx, robotID, v1alpha1robot.RobotsRefreshRobotAccountTokenBody{
				AutoGenerate: true,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Token).NotTo(BeEmpty())
			Expect(resp.Token).NotTo(Equal(initialToken), "refreshed token must differ from the initial one")

			GinkgoWriter.Printf("Token refreshed successfully\n")
		})

		It("should fail to refresh token for a non-existent robot", Label("R00014"), func() {
			_, _, err := robotsApi.RobotsRefreshRobotAccountToken(ctx, 999999999, v1alpha1robot.RobotsRefreshRobotAccountTokenBody{
				AutoGenerate: true,
			})
			Expect(err).To(HaveOccurred(), "refreshing token for non-existent robot must return an error")
		})

		It("should accept a valid manually specified token", Label("R00016"), func() {
			manualToken := "RobotToken9x"
			resp, _, err := robotsApi.RobotsRefreshRobotAccountToken(ctx, robotID, v1alpha1robot.RobotsRefreshRobotAccountTokenBody{
				AutoGenerate: false,
				Token:        manualToken,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Token).To(Equal(manualToken))
		})
	})

	// ═══════════════════════════════════════════════════════════
	// 5. Robot token authentication and status lifecycle
	// ═══════════════════════════════════════════════════════════
	Context("Robot token authentication", func() {
		var (
			robotName    string
			robotID      int64
			initialToken string
			usersAPI     *v1alpha1user.UsersApiService
		)

		BeforeEach(func() {
			robotName = generateRobotName("rb-auth")
			createResp, _, err := robotsApi.RobotsCreateRobotAccount(ctx, v1alpha1robot.V1alpha1CreateRobotAccountRequest{
				Name:                robotName,
				PlatformPermissions: []string{"user.create"},
			})
			Expect(err).NotTo(HaveOccurred())
			initialToken = createResp.Token

			listResp, _, err := robotsApi.RobotsListRobotAccounts(ctx, &v1alpha1robot.RobotsApiRobotsListRobotAccountsOpts{
				Search: optional.NewString(robotPrefix + robotName),
			})
			Expect(err).NotTo(HaveOccurred())
			for _, rb := range listResp.Items {
				if rb.Name == robotPrefix+robotName {
					robotID = rb.Id
					break
				}
			}
			Expect(robotID).NotTo(BeZero())
			usersAPI = newRobotUsersAPI()
		})

		AfterEach(func() {
			if robotID != 0 {
				_, _, _ = robotsApi.RobotsDeleteRobotAccount(ctx, robotID)
			}
		})

		createUser := func(token, prefix string) (string, error) {
			username := tools.GenerateTestUsername(prefix)
			_, _, err := usersAPI.UsersCreateUser(
				robotAuthContext(robotPrefix+robotName, token),
				v1alpha1user.V1alpha1CreateUserRequest{
					Username: username,
					Password: "Test@123456",
				},
			)
			return username, err
		}

		cleanupUser := func(username string) {
			userID, err := tools.GetUserIDByUsername(username)
			if err == nil {
				_ = tools.DeleteUser(userID)
			}
		}

		It("should authenticate with the generated token and enforce token validity", Label("R00017"), func() {
			createdUsername, err := createUser(initialToken, "rb-auth-ok")
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(cleanupUser, createdUsername)

			_, err = createUser("wrong-robot-token", "rb-auth-bad")
			Expect(err).To(HaveOccurred(), "an invalid robot token must be rejected")
		})

		It("should invalidate the old token after refresh", Label("R00018"), func() {
			refreshResp, _, err := robotsApi.RobotsRefreshRobotAccountToken(ctx, robotID, v1alpha1robot.RobotsRefreshRobotAccountTokenBody{
				AutoGenerate: true,
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = createUser(initialToken, "rb-old-token")
			Expect(err).To(HaveOccurred(), "the previous token must stop working immediately")

			createdUsername, err := createUser(refreshResp.Token, "rb-new-token")
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(cleanupUser, createdUsername)
		})

		It("should disable and re-enable API access", Label("R00019"), func() {
			_, _, err := robotsApi.RobotsUpdateRobotAccount(ctx, robotID, v1alpha1robot.RobotsUpdateRobotAccountBody{
				Status:              robotStatus(v1alpha1robot.DISABLED_V1alpha1RobotAccountStatus),
				PlatformPermissions: []string{"user.create"},
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = createUser(initialToken, "rb-disabled")
			Expect(err).To(HaveOccurred(), "a disabled robot must not authenticate")

			_, _, err = robotsApi.RobotsUpdateRobotAccount(ctx, robotID, v1alpha1robot.RobotsUpdateRobotAccountBody{
				Status:              robotStatus(v1alpha1robot.ENABLED_V1alpha1RobotAccountStatus),
				PlatformPermissions: []string{"user.create"},
			})
			Expect(err).NotTo(HaveOccurred())

			createdUsername, err := createUser(initialToken, "rb-enabled")
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(cleanupUser, createdUsername)
		})
	})

	// ═══════════════════════════════════════════════════════════
	// 6. DeleteRobotAccount API
	// ═══════════════════════════════════════════════════════════
	Context("DeleteRobotAccount API", func() {
		It("should delete a robot account and make it inaccessible", Label("R00015"), func() {
			name := generateRobotName("rb-del")
			_, _, err := robotsApi.RobotsCreateRobotAccount(ctx, v1alpha1robot.V1alpha1CreateRobotAccountRequest{
				Name: name,
			})
			Expect(err).NotTo(HaveOccurred())

			listResp, _, err := robotsApi.RobotsListRobotAccounts(ctx, &v1alpha1robot.RobotsApiRobotsListRobotAccountsOpts{
				Search: optional.NewString(robotPrefix + name),
			})
			Expect(err).NotTo(HaveOccurred())
			var robotID int64
			for _, r := range listResp.Items {
				if r.Name == robotPrefix+name {
					robotID = r.Id
					break
				}
			}
			Expect(robotID).NotTo(BeZero())

			_, _, err = robotsApi.RobotsDeleteRobotAccount(ctx, robotID)
			Expect(err).NotTo(HaveOccurred())

			_, _, err = robotsApi.RobotsGetRobotAccount(ctx, robotID)
			Expect(err).To(HaveOccurred(), "deleted robot must no longer be accessible")
		})
	})
})
