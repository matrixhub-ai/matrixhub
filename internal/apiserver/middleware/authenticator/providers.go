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
	"github.com/matrixhub-ai/matrixhub/internal/domain/robot"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
)

func NewWebAuthenticator(sessionRepo user.ISessionRepo, tokenRepo user.IAccessTokenRepo, robotRepo robot.IRobotRepo) *MultiAuthenticator {
	return NewMultiAuthenticator(
		NewCookieAuthenticator(sessionRepo),
		NewTokenAuthenticator(tokenRepo),
		NewRobotAuthenticator(robotRepo),
	)
}

func NewHfCLIAuthenticator(tokenRepo user.IAccessTokenRepo, sessionRepo user.ISessionRepo, robotRepo robot.IRobotRepo) *MultiAuthenticator {
	return NewMultiAuthenticator(
		NewTokenAuthenticator(tokenRepo),
		NewCookieAuthenticator(sessionRepo),
		NewRobotTokenAuthenticator(robotRepo),
	)
}

func NewGitAuthenticator(tokenRepo user.IAccessTokenRepo, robotRepo robot.IRobotRepo) *MultiAuthenticator {
	return NewMultiAuthenticator(
		NewTokenAuthenticator(tokenRepo),
		NewRobotTokenAuthenticator(robotRepo),
	)
}
