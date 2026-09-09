package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/matrixhub-ai/hfd/pkg/authenticate"
	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/domain/robot"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

func TestGitAuthRejectsInvalidCredentials(t *testing.T) {
	expired := time.Now().Add(-time.Hour)
	for _, test := range []struct {
		name   string
		token  string
		access *user.AccessToken
		robot  *robot.Robot
		err    error
		status int
	}{
		{"unrecognized", "forged", nil, nil, nil, http.StatusUnauthorized},
		{"deleted", utils.TokenPrefix + "deleted", &user.AccessToken{}, nil, nil, http.StatusUnauthorized},
		{"expired", utils.TokenPrefix + "expired", &user.AccessToken{Enabled: true, ExpireAt: &expired}, nil, nil, http.StatusUnauthorized},
		{"missing robot", utils.RobotTokenPrefix + "missing", nil, nil, gorm.ErrRecordNotFound, http.StatusUnauthorized},
		{"disabled robot", utils.RobotTokenPrefix + "disabled", nil, &robot.Robot{}, nil, http.StatusUnauthorized},
		{"storage failure", utils.TokenPrefix + "unavailable", nil, nil, errors.New("database unavailable"), http.StatusInternalServerError},
	} {
		for _, scheme := range []string{"basic", "bearer"} {
			t.Run(test.name+"/"+scheme, func(t *testing.T) {
				accessRepo := &gitAuthAccessRepo{token: test.access, err: test.err}
				robotRepo := &gitAuthRobotRepo{robot: test.robot, err: test.err}
				handler := authenticate.NewHandler(
					authenticate.WithBasicAuthValidator(GitBasicAuthAuthn(accessRepo, nil, robotRepo)),
					authenticate.WithTokenValidator(GitHTTPAuthn(accessRepo, nil, robotRepo)),
					authenticate.WithNext(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
						t.Error("invalid credential reached the next handler")
					})),
				)
				request := httptest.NewRequest(http.MethodGet, "/api/whoami-v2", nil)
				if scheme == "basic" {
					request.SetBasicAuth("alice", test.token)
				} else {
					request.Header.Set("Authorization", "Bearer "+test.token)
				}
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if response.Code != test.status {
					t.Errorf("status = %d, want %d", response.Code, test.status)
				}
			})
		}
	}
}

type gitAuthAccessRepo struct {
	user.IAccessTokenRepo
	token *user.AccessToken
	err   error
}

func (repo *gitAuthAccessRepo) GetByTokenHash(context.Context, string) (*user.AccessToken, error) {
	return repo.token, repo.err
}

type gitAuthRobotRepo struct {
	robot.IRobotRepo
	robot *robot.Robot
	err   error
}

func (repo *gitAuthRobotRepo) GetRobotByTokenHash(context.Context, string) (*robot.Robot, error) {
	return repo.robot, repo.err
}
