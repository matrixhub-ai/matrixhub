package hfd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/matrixhub-ai/hfd/pkg/authenticate"
	"github.com/matrixhub-ai/hfd/pkg/permission"

	"github.com/matrixhub-ai/matrixhub/internal/apiserver/middleware"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/config"
)

func TestHandlerGitAuthenticationChallenge(t *testing.T) {
	backend := New(&config.Config{DataDir: t.TempDir(), APIServer: &config.APIServerConfig{TokenSigningSecret: "test-secret"}})
	backend.sessionRepo = &handlerSessionRepo{manager: scs.New()}
	backend.permissionHookFunc = func(context.Context, permission.Operation, string, permission.Context) (bool, error) {
		return false, nil
	}
	handler := backend.Handler(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusForbidden)
	}))

	for _, test := range []struct {
		name      string
		method    string
		url       string
		user      string
		status    int
		challenge string
	}{
		{"anonymous clone", http.MethodGet, "/private/model.git/info/refs?service=git-upload-pack", "", http.StatusUnauthorized, `Basic realm="hfd"`},
		{"anonymous push discovery", http.MethodGet, "/private/model/info/refs?service=git-receive-pack", "", http.StatusUnauthorized, `Basic realm="hfd"`},
		{"anonymous fetch", http.MethodPost, "/private/model.git/git-upload-pack", "", http.StatusUnauthorized, `Basic realm="hfd"`},
		{"anonymous push", http.MethodPost, "/private/model/git-receive-pack", "", http.StatusUnauthorized, `Basic realm="hfd"`},
		{"authenticated denial", http.MethodGet, "/private/model.git/info/refs?service=git-upload-pack", "alice", http.StatusForbidden, ""},
		{"invalid service", http.MethodGet, "/private/model.git/info/refs?service=git-invalid", "", http.StatusForbidden, ""},
		{"downstream API", http.MethodGet, "/api/v1/test", "", http.StatusForbidden, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.url, nil)
			if test.user != "" {
				request = request.WithContext(authenticate.WithContext(request.Context(), authenticate.UserInfo{User: test.user}))
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Errorf("status = %d, want %d", response.Code, test.status)
			}
			if got := response.Header().Get("WWW-Authenticate"); got != test.challenge {
				t.Errorf("challenge = %q, want %q", got, test.challenge)
			}
		})
	}
}

func TestHandlerCASTokenCannotAuthorizeRepository(t *testing.T) {
	backend := New(&config.Config{DataDir: t.TempDir(), APIServer: &config.APIServerConfig{TokenSigningSecret: "test-secret"}})
	backend.sessionRepo = &handlerSessionRepo{manager: scs.New()}
	backend.permissionHookFunc = normalizePermissionHook(middleware.NewRepoEnforcer(nil))
	mint, _, err := authenticate.NewXETTokenScheme(backend.auth.tokenSignValidator)
	if err != nil {
		t.Fatal(err)
	}
	token, _ := mint(time.Now())
	request := httptest.NewRequest(http.MethodGet, "/private/model.git/info/refs?service=git-upload-pack", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	backend.Handler(http.NotFoundHandler()).ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
}

type handlerSessionRepo struct {
	user.ISessionRepo
	manager *scs.SessionManager
}

func (repo *handlerSessionRepo) Manager() *scs.SessionManager {
	return repo.manager
}
