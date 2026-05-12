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
	"strings"

	"github.com/antihax/optional"
	v1alpha1robot "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/robot"
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
		It("should create a robot account and return a non-empty token", Label("R00001"), func() {
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
			})
			Expect(err).NotTo(HaveOccurred())

			resp, _, err := robotsApi.RobotsGetRobotAccount(ctx, robotID)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Description).To(Equal("updated description"))

			GinkgoWriter.Printf("Updated robot description: %s\n", resp.Description)
		})

		It("should update the robot to never-expire when expire_days is set to 0", Label("R00011"), func() {
			// First create with expiry.
			_, _, err := robotsApi.RobotsUpdateRobotAccount(ctx, robotID, v1alpha1robot.RobotsUpdateRobotAccountBody{
				ExpireDays: 0,
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
	})

	// ═══════════════════════════════════════════════════════════
	// 5. DeleteRobotAccount API
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

