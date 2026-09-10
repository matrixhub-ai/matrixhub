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

package authenticator

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/matrixhub-ai/matrixhub/internal/domain/auth"
	"github.com/matrixhub-ai/matrixhub/internal/domain/robot"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

type RobotAuthenticator struct {
	robotRepo robot.IRobotRepo
}

func NewRobotAuthenticator(robotRepo robot.IRobotRepo) *RobotAuthenticator {
	return &RobotAuthenticator{robotRepo: robotRepo}
}

func (a *RobotAuthenticator) Authenticate(ctx context.Context, r *http.Request) (auth.Identity, bool, bool, error) {
	username, secret, err := utils.ParseBasicAuthFromGRPCContext(ctx)
	if err != nil {
		return nil, true, false, nil
	}
	return a.AuthenticateToken(ctx, username, secret)
}

func (a *RobotAuthenticator) AuthenticateToken(ctx context.Context, username, token string) (auth.Identity, bool, bool, error) {
	if !strings.HasPrefix(username, robot.RobotPrefix) {
		return nil, true, false, nil
	}
	rb, err := a.robotRepo.GetRobotByName(ctx, username)
	if err != nil {
		return nil, false, false, err
	}
	if !rb.IsValid(time.Now()) {
		return nil, false, false, nil
	}
	if !rb.CheckTokenHash(utils.Sha256Hex(token)) {
		return nil, false, false, nil
	}

	return robot.NewRobotIdentity(rb.ID, rb.Name), false, true, nil
}
