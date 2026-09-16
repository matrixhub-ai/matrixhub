package hfd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/matrixhub-ai/hfd/pkg/authenticate"
	"github.com/matrixhub-ai/hfd/pkg/permission"
	"github.com/matrixhub-ai/hfd/pkg/repository"
	xetauth "github.com/wzshiming/xet/auth"

	"github.com/matrixhub-ai/matrixhub/internal/apiserver/middleware"
	"github.com/matrixhub-ai/matrixhub/internal/domain/auth"
	"github.com/matrixhub-ai/matrixhub/internal/domain/authz"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
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
				request = request.WithContext(authenticate.WithIdentity(request.Context(), middleware.Principal{Identity: user.NewUserIdentity(1, test.user)}))
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

func TestHandlerCASTokenIsNotAUser(t *testing.T) {
	backend := New(&config.Config{DataDir: t.TempDir(), APIServer: &config.APIServerConfig{TokenSigningSecret: "test-secret"}})
	backend.sessionRepo = &handlerSessionRepo{manager: scs.New()}
	var seen authenticate.Identity
	backend.permissionHookFunc = func(ctx context.Context, _ permission.Operation, _ string, _ permission.Context) (bool, error) {
		seen = authenticate.IdentityFrom(ctx)
		return false, nil
	}
	token, _, err := backend.storage.casIssuer.Sign(xetauth.Grant{Permission: xetauth.Read})
	if err != nil {
		t.Fatal(err)
	}
	handler := backend.Handler(http.NotFoundHandler())
	request := httptest.NewRequest(http.MethodGet, "/private/model.git/info/refs?service=git-upload-pack", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.Code)
	}
	if got := response.Header().Get("WWW-Authenticate"); got != `Basic realm="hfd"` {
		t.Errorf("challenge = %q, want Basic realm=\"hfd\"", got)
	}
	if seen == nil || !authenticate.IsAnonymous(seen) {
		t.Errorf("permission hook identity = %v, want anonymous", seen)
	}

	url := "/v1/reconstructions/" + strings.Repeat("ab", 32)
	request = httptest.NewRequest(http.MethodGet, url, nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("authenticated CAS status = %d, want 404", response.Code)
	}

	request = httptest.NewRequest(http.MethodGet, url, nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous CAS status = %d, want 401", response.Code)
	}
}

func TestHandlerListAuthorization(t *testing.T) {
	backend := New(&config.Config{DataDir: t.TempDir(), APIServer: &config.APIServerConfig{TokenSigningSecret: "test-secret"}})
	backend.sessionRepo = &handlerSessionRepo{manager: scs.New()}
	backend.permissionHookFunc = middleware.NewRepoEnforcer(handlerAuthzService{})
	for _, path := range []string{"/pub/model.git", "/priv/model.git", "/datasets/pub/data.git", "/datasets/priv/data.git"} {
		if _, err := repository.Init(context.Background(), backend.Storage().RepositoriesFS(), path, "main"); err != nil {
			t.Fatal(err)
		}
	}
	handler := backend.Handler(http.NotFoundHandler())

	for _, test := range []struct {
		name   string
		url    string
		userID int
		status int
		ids    string
	}{
		{"anonymous public models", "/api/models?author=pub", 0, http.StatusOK, "pub/model"},
		{"anonymous public datasets", "/api/datasets?author=pub", 0, http.StatusOK, "pub/data"},
		{"member private models", "/api/models?author=priv", 1, http.StatusOK, "priv/model"},
		{"anonymous private models", "/api/models?author=priv", 0, http.StatusForbidden, ""},
		{"other user private datasets", "/api/datasets?author=priv", 2, http.StatusForbidden, ""},
		{"member without author", "/api/models", 1, http.StatusForbidden, ""},
		{"spaces", "/api/spaces?author=pub", 0, http.StatusForbidden, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.url, nil)
			if test.userID != 0 {
				request = request.WithContext(authenticate.WithIdentity(request.Context(), middleware.Principal{Identity: user.NewUserIdentity(test.userID, "user")}))
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.status, response.Body.String())
			}
			if strings.Contains(response.Body.String(), "priv/") != strings.HasPrefix(test.ids, "priv/") {
				t.Fatalf("private namespace disclosure: body = %s", response.Body.String())
			}
			if test.status != http.StatusOK {
				return
			}
			var items []struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &items); err != nil {
				t.Fatal(err)
			}
			var ids []string
			for _, item := range items {
				ids = append(ids, item.ID)
			}
			if got := strings.Join(ids, ","); got != test.ids {
				t.Errorf("ids = %q, want %q", got, test.ids)
			}
		})
	}
}

type handlerAuthzService struct {
	authz.IAuthzService
}

func (handlerAuthzService) VerifyProjectPermissionByName(ctx context.Context, project string, perm role.Permission) (bool, error) {
	if project == "pub" {
		return perm == role.ModelPull || perm == role.DatasetPull, nil
	}
	identity, ok := auth.IdentityFromContext(ctx)
	return project == "priv" && ok && identity.GetID() == 1, nil
}

type handlerSessionRepo struct {
	user.ISessionRepo
	manager *scs.SessionManager
}

func (repo *handlerSessionRepo) Manager() *scs.SessionManager {
	return repo.manager
}
