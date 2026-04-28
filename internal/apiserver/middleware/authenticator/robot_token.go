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
	"errors"
	"net/http"
	"strings"

	"github.com/matrixhub-ai/matrixhub/internal/domain/auth"
	"github.com/matrixhub-ai/matrixhub/internal/domain/robot"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

type RobotTokenAuthenticator struct {
	robotRepo robot.IRobotRepo
}

func NewRobotTokenAuthenticator(robotRepo robot.IRobotRepo) *RobotTokenAuthenticator {
	return &RobotTokenAuthenticator{robotRepo: robotRepo}
}

func (a *RobotTokenAuthenticator) Authenticate(ctx context.Context, r *http.Request) (auth.Identity, error) {
	token := extractTokenCredential(ctx, r)
	return a.AuthenticateToken(ctx, "", token)
}

func (a *RobotTokenAuthenticator) AuthenticateToken(ctx context.Context, _, token string) (auth.Identity, error) {
	if token == "" || !strings.HasPrefix(token, utils.RobotTokenPrefix) {
		return nil, nil
	}

	rb, err := a.robotRepo.GetRobotByTokenHash(ctx, utils.Sha256Hex(token))
	if err != nil {
		return nil, err
	}
	if rb == nil || !rb.Enabled {
		return nil, errors.New("robot is invalid")
	}

	return robot.NewRobotIdentity(rb.ID, rb.Name), nil
}
