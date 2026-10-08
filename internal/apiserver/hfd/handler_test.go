package hfd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/matrixhub-ai/hfd/pkg/authenticate"
	backendhttp "github.com/matrixhub-ai/hfd/pkg/backend/http"
	"github.com/matrixhub-ai/hfd/pkg/mirror"
	"github.com/matrixhub-ai/hfd/pkg/permission"
	"github.com/matrixhub-ai/hfd/pkg/repository"
	gitstorage "github.com/matrixhub-ai/hfd/pkg/storage"
	xetauth "github.com/wzshiming/xet/auth"
	"go.uber.org/mock/gomock"
	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/apiserver/middleware"
	"github.com/matrixhub-ai/matrixhub/internal/domain/auth"
	"github.com/matrixhub-ai/matrixhub/internal/domain/authz"
	"github.com/matrixhub-ai/matrixhub/internal/domain/dataset"
	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
	modelmocks "github.com/matrixhub-ai/matrixhub/internal/domain/model/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	projectmocks "github.com/matrixhub-ai/matrixhub/internal/domain/project/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	registrymocks "github.com/matrixhub-ai/matrixhub/internal/domain/registry/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/robot"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/config"
)

func TestHandlerGitAuthenticationChallenge(t *testing.T) {
	backend, _ := New(t.Context(), &config.Config{DataDir: t.TempDir(), APIServer: &config.APIServerConfig{TokenSigningSecret: "test-secret"}})
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

// HF clients connect to the minted casUrl directly, so it must be the origin
// they reach the server at, not the loopback hostURL config.Init defaults to.
func TestHandlerCASURLIsReachableByTheClient(t *testing.T) {
	for _, test := range []struct {
		name        string
		externalURL string
		want        string
	}{
		{"request origin", "", "https://hub.example.com"},
		{"configured external URL", "https://public.example.com/", "https://public.example.com"},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := testConfig(t.TempDir())
			cfg.APIServer.HostURL, cfg.APIServer.ExternalURL = "http://localhost:3001", test.externalURL
			backend, err := New(t.Context(), cfg)
			if err != nil {
				t.Fatal(err)
			}
			backend.sessionRepo = &handlerSessionRepo{manager: scs.New()}
			backend.permissionHookFunc = func(context.Context, permission.Operation, string, permission.Context) (bool, error) {
				return true, nil
			}
			request := httptest.NewRequest(http.MethodGet, "https://hub.example.com/api/models/proj/model/xet-read-token/main", nil)
			response := httptest.NewRecorder()
			backend.Handler(http.NotFoundHandler()).ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body = %s", response.Code, response.Body.String())
			}
			if got := response.Header().Get("X-Xet-Cas-Url"); got != test.want {
				t.Fatalf("X-Xet-Cas-Url = %q, want %q", got, test.want)
			}
		})
	}
}

func TestHandlerCASTokenIsNotAUser(t *testing.T) {
	backend, _ := New(t.Context(), &config.Config{DataDir: t.TempDir(), APIServer: &config.APIServerConfig{TokenSigningSecret: "test-secret"}})
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
	backend, _ := New(t.Context(), &config.Config{DataDir: t.TempDir(), APIServer: &config.APIServerConfig{TokenSigningSecret: "test-secret"}})
	backend.sessionRepo = &handlerSessionRepo{manager: scs.New()}
	backend.permissionHookFunc = middleware.NewRepoEnforcer(handlerAuthzService{})
	ctrl := gomock.NewController(t)
	projectRepo := projectmocks.NewMockIProjectRepo(ctrl)
	projectRepo.EXPECT().GetProjectByName(gomock.Any(), "pub").Return(&project.Project{Name: "pub", Type: project.ProjectTypePublic}, nil).Times(2)
	projectRepo.EXPECT().GetProjectByName(gomock.Any(), "priv").Return(&project.Project{Name: "priv"}, nil)
	modelService := modelmocks.NewMockIModelService(ctrl)
	for _, name := range []string{"pub", "priv"} {
		modelService.EXPECT().ListModels(gomock.Any(), &model.Filter{Project: name, Page: 1, PageSize: 100}).Return([]*model.Model{{ProjectName: name, Name: "model"}}, int64(1), nil)
	}
	datasetService := &handlerDatasetService{lists: []*model.Filter{{Project: "pub", Page: 1, PageSize: 100}}, rows: []*dataset.Dataset{{ProjectName: "pub", Name: "data"}}, total: 1}
	backend.projectRepo, backend.modelService, backend.datasetService = projectRepo, modelService, datasetService
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
	if len(datasetService.lists) != 0 {
		t.Errorf("dataset filters not consumed: %+v", datasetService.lists)
	}
}

type handlerAuthzService struct {
	authz.IAuthzService
}

func TestHandlerWhoami(t *testing.T) {
	dbErr := errors.New("database offline")
	expired := time.Now().Add(-time.Hour)
	const tail = `"emailVerified":false,"isPro":false,"canPay":false,"orgs":[%s],"auth":{"accessToken":{"displayName":"token","role":"write"}}}`
	const alice = `{"type":"user","id":"1","name":"alice","fullname":"alice","email":"alice@example.com",` + tail
	const bot = `{"type":"robot","id":"2","name":"bot","fullname":"bot",` + tail
	org := func(id int, name string) string {
		return fmt.Sprintf(`{"type":"org","id":"%d","name":"%[2]s","fullname":"%[2]s"}`, id, name)
	}
	for _, test := range []struct {
		name         string
		identity     auth.Identity
		users        *handlerWhoamiUserRepo
		robots       *handlerWhoamiRobotRepo
		projectNames []string
		projects     []*project.Project
		status       int
		body         string
	}{
		{"user", user.NewUserIdentity(1, "stale-alice"),
			&handlerWhoamiUserRepo{account: &user.User{ID: 1, Username: "alice", Email: "alice@example.com"}, roles: map[string]int{"zeta": 2, "alpha": 1}}, nil,
			[]string{"alpha", "zeta"}, []*project.Project{{ID: 9, Name: "zeta"}, {ID: 4, Name: "alpha"}},
			http.StatusOK, fmt.Sprintf(alice, org(4, "alpha")+","+org(9, "zeta"))},
		{"user without projects", user.NewUserIdentity(1, "alice"),
			&handlerWhoamiUserRepo{account: &user.User{ID: 1, Username: "alice", Email: "alice@example.com"}}, nil, nil, nil,
			http.StatusOK, fmt.Sprintf(alice, "")},
		{"robot", robot.NewRobotIdentity(2, "stale-bot"), nil,
			&handlerWhoamiRobotRepo{account: &robot.Robot{ID: 2, Name: "bot", Enabled: true, Projects: []*project.Project{{ID: 5, Name: "proj"}}}}, nil, nil,
			http.StatusOK, fmt.Sprintf(bot, org(5, "proj"))},
		{"user missing", user.NewUserIdentity(1, "alice"), &handlerWhoamiUserRepo{}, nil, nil, nil, http.StatusInternalServerError, `{"error":"whoami user 1: not found"}`},
		{"user query error", user.NewUserIdentity(1, "alice"), &handlerWhoamiUserRepo{err: dbErr}, nil, nil, nil, http.StatusInternalServerError, `{"error":"whoami user 1: database offline"}`},
		{"robot missing", robot.NewRobotIdentity(2, "bot"), nil, &handlerWhoamiRobotRepo{err: dbErr}, nil, nil, http.StatusInternalServerError, `{"error":"whoami robot 2: database offline"}`},
		{"robot disabled", robot.NewRobotIdentity(2, "bot"), nil, &handlerWhoamiRobotRepo{account: &robot.Robot{ID: 2, Name: "bot"}}, nil, nil, http.StatusInternalServerError, `{"error":"whoami robot 2: not found or disabled"}`},
		{"robot expired", robot.NewRobotIdentity(2, "bot"), nil, &handlerWhoamiRobotRepo{account: &robot.Robot{ID: 2, Name: "bot", Enabled: true, ExpireAt: &expired}}, nil, nil, http.StatusInternalServerError, `{"error":"whoami robot 2: not found or disabled"}`},
		{"anonymous", nil, &handlerWhoamiUserRepo{err: dbErr}, &handlerWhoamiRobotRepo{err: dbErr}, nil, nil, http.StatusUnauthorized, `{"error":"Unauthorized"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			backend, err := New(t.Context(), testConfig(t.TempDir()))
			if err != nil {
				t.Fatal(err)
			}
			backend.sessionRepo = &handlerSessionRepo{manager: scs.New()}
			projectRepo := projectmocks.NewMockIProjectRepo(gomock.NewController(t))
			if test.projectNames != nil {
				projectRepo.EXPECT().ListProjectInfoByNames(gomock.Any(), test.projectNames).Return(test.projects, nil)
			}
			backend.projectRepo = projectRepo
			if test.users != nil {
				backend.userRepo = test.users
			}
			if test.robots != nil {
				backend.robotRepo = test.robots
			}
			request := httptest.NewRequest(http.MethodGet, "/api/whoami-v2", nil)
			if test.identity != nil {
				request = request.WithContext(authenticate.WithIdentity(request.Context(), middleware.Principal{Identity: test.identity}))
			}
			response := httptest.NewRecorder()
			backend.Handler(http.NotFoundHandler()).ServeHTTP(response, request)
			if response.Code != test.status || strings.TrimSpace(response.Body.String()) != test.body {
				t.Fatalf("status = %d, body = %s; want %d, %s", response.Code, response.Body.String(), test.status, test.body)
			}
			wantLookup := 0
			if test.identity != nil {
				wantLookup = test.identity.GetID()
			}
			if test.users != nil && test.users.lookedUpID != wantLookup {
				t.Errorf("user lookup = %d, want %d", test.users.lookedUpID, wantLookup)
			}
			if test.robots != nil && test.robots.lookedUpID != wantLookup {
				t.Errorf("robot lookup = %d, want %d", test.robots.lookedUpID, wantLookup)
			}
		})
	}
}

func TestHandlerWhoamiReadsDatabase(t *testing.T) {
	backend, err := New(t.Context(), testConfig(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	backend.sessionRepo = &handlerSessionRepo{manager: scs.New()}
	users := &handlerWhoamiUserRepo{account: &user.User{ID: 42, Username: "db-alice", Email: "alice@example.com"}}
	backend.userRepo = users
	request := httptest.NewRequest(http.MethodGet, "/api/whoami-v2", nil)
	request = request.WithContext(authenticate.WithIdentity(request.Context(), middleware.Principal{Identity: user.NewUserIdentity(42, "old-alice")}))
	response := httptest.NewRecorder()
	backend.Handler(http.NotFoundHandler()).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s; want 200", response.Code, response.Body.String())
	}
	var account struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &account); err != nil {
		t.Fatal(err)
	}
	if users.lookedUpID != 42 || account.ID != "42" || account.Name != users.account.Username || account.Email != users.account.Email {
		t.Fatalf("lookup ID = %d, account = %+v; want DB user 42/db-alice/alice@example.com", users.lookedUpID, account)
	}
}

type handlerWhoamiUserRepo struct {
	user.IUserRepo
	account    *user.User
	err        error
	roles      map[string]int
	lookedUpID int
}

func (repo *handlerWhoamiUserRepo) GetUser(_ context.Context, id int) (*user.User, error) {
	repo.lookedUpID = id
	return repo.account, repo.err
}

func (repo *handlerWhoamiUserRepo) GetUserAllProjectRoles(context.Context, int) (map[string]int, error) {
	return repo.roles, nil
}

type handlerWhoamiRobotRepo struct {
	robot.IRobotRepo
	account    *robot.Robot
	err        error
	lookedUpID int
}

func (repo *handlerWhoamiRobotRepo) GetRobot(_ context.Context, id int) (*robot.Robot, error) {
	repo.lookedUpID = id
	return repo.account, repo.err
}

func TestHandlerListRepos(t *testing.T) {
	labels := []model.Label{{Name: "llama", Category: "other"}, {Name: "text-generation", Category: "task"}, {Name: "transformers", Category: "library"}, {Name: "qwen2", Category: "other"}}
	models := []*model.Model{{ID: 1, ProjectName: "pub", Name: "alpha", Labels: labels}, {ID: 2, ProjectName: "pub", Name: "beta"}, {ID: 3, ProjectName: "pub", Name: "beta-2"}}
	public := &project.Project{ID: 1, Name: "pub", Type: project.ProjectTypePublic}
	private := &project.Project{ID: 2, Name: "priv", Type: project.ProjectTypePrivate}
	dbErr := errors.New("database offline")
	type call struct {
		filter *model.Filter
		rows   []*model.Model
		total  int64
		err    error
	}
	filter := func(project string, page, pageSize int32, search string, labels []string) *model.Filter {
		return &model.Filter{Project: project, Search: search, Label: labels, Page: page, PageSize: pageSize}
	}
	cursor := func(offset int) string {
		return base64.URLEncoding.EncodeToString(fmt.Appendf(nil, `{"offset":%d}`, offset))
	}
	link := func(offset int) string {
		return fmt.Sprintf(`<http://example.com/api/models?author=pub&cursor=%s&limit=2>; rel="next"`, cursor(offset))
	}
	const alphaItem = `[{"id":"pub/alpha","likes":0,"trendingScore":0,"private":false,"downloads":0,"tags":["llama","text-generation","transformers","qwen2"],"pipeline_tag":"text-generation","library_name":"transformers","modelId":"pub/alpha"}]`
	const secretItem = `[{"id":"priv/secret","likes":0,"trendingScore":0,"private":true,"downloads":0,"modelId":"priv/secret"}]`
	const dataItem = `[{"id":"pub/data","likes":0,"trendingScore":0,"private":false,"downloads":0}]`
	for _, test := range []struct {
		name       string
		path       string
		project    *project.Project
		projectErr error
		calls      []call
		datasets   bool
		status     int
		ids        string
		body       string
		link       string
	}{
		{"author models", "/api/models?author=pub", public, nil, []call{{filter("pub", 1, 100, "", nil), models, 3, nil}}, false, http.StatusOK, "pub/alpha,pub/beta,pub/beta-2", "", ""},
		{"private project", "/api/models?author=priv", private, nil, []call{{filter("priv", 1, 100, "", nil), []*model.Model{{ProjectName: "priv", Name: "secret"}}, 1, nil}}, false, http.StatusOK, "priv/secret", secretItem, ""},
		{"unknown author", "/api/models?author=nobody", nil, gorm.ErrRecordNotFound, nil, false, http.StatusOK, "", "[]", ""},
		{"project query error", "/api/models?author=pub", nil, dbErr, nil, false, http.StatusInternalServerError, "", `{"error":"database offline"}`, ""},
		{"no author", "/api/models", nil, nil, nil, false, http.StatusOK, "", "[]", ""},
		{"nested author", "/api/models?author=../datasets/pub", nil, nil, nil, false, http.StatusOK, "", "[]", ""},
		{"spaces", "/api/spaces?author=pub", nil, nil, nil, false, http.StatusOK, "", "[]", ""},
		{"search", "/api/models?author=pub&search=ALP", public, nil, []call{{filter("pub", 1, 100, "ALP", nil), models[:1], 1, nil}}, false, http.StatusOK, "pub/alpha", alphaItem, ""},
		{"tags", "/api/models?author=pub&filter=text-generation&filter=qwen2", public, nil, []call{{filter("pub", 1, 100, "", []string{"text-generation", "qwen2"}), models[:1], 1, nil}}, false, http.StatusOK, "pub/alpha", alphaItem, ""},
		{"metric sort keeps DB order", "/api/models?author=pub&sort=downloads", public, nil, []call{{filter("pub", 1, 100, "", nil), models, 3, nil}}, false, http.StatusOK, "pub/alpha,pub/beta,pub/beta-2", "", ""},
		{"first page", "/api/models?author=pub&limit=2", public, nil, []call{{filter("pub", 1, 2, "", nil), models[:2], 3, nil}}, false, http.StatusOK, "pub/alpha,pub/beta", "", link(2)},
		{"second page", "/api/models?author=pub&cursor=" + cursor(2) + "&limit=2", public, nil, []call{{filter("pub", 2, 2, "", nil), models[2:], 3, nil}}, false, http.StatusOK, "pub/beta-2", "", ""},
		{"unaligned cursor", "/api/models?author=pub&cursor=" + cursor(1) + "&limit=2", public, nil, []call{{filter("pub", 1, 2, "", nil), models[:2], 3, nil}, {filter("pub", 2, 2, "", nil), models[2:], 3, nil}}, false, http.StatusOK, "pub/beta,pub/beta-2", "", ""},
		{"unaligned cursor with more", "/api/models?author=pub&cursor=" + cursor(1) + "&limit=2", public, nil, []call{{filter("pub", 1, 2, "", nil), models[:2], 4, nil}, {filter("pub", 2, 2, "", nil), []*model.Model{models[2], {ProjectName: "pub", Name: "gamma"}}, 4, nil}}, false, http.StatusOK, "pub/beta,pub/beta-2", "", link(3)},
		{"cursor past end", "/api/models?author=pub&cursor=" + cursor(5) + "&limit=2", public, nil, []call{{filter("pub", 3, 2, "", nil), nil, 3, nil}}, false, http.StatusOK, "", "[]", ""},
		{"cursor beyond DB offset", "/api/models?author=pub&cursor=" + cursor(math.MaxInt32+1) + "&limit=2", nil, nil, nil, false, http.StatusOK, "", "[]", ""},
		{"dataset cursor beyond DB offset", "/api/datasets?author=pub&cursor=" + cursor(math.MaxInt32+1) + "&limit=2", nil, nil, nil, true, http.StatusOK, "", "[]", ""},
		{"maximum cursor", "/api/models?author=pub&cursor=" + cursor(math.MaxInt) + "&limit=1", nil, nil, nil, false, http.StatusOK, "", "[]", ""},
		{"maximum limit", fmt.Sprintf("/api/models?author=pub&limit=%d", math.MaxInt), public, nil, []call{{filter("pub", 1, math.MaxInt32, "", nil), models, 3, nil}}, false, http.StatusOK, "pub/alpha,pub/beta,pub/beta-2", "", ""},
		{"last DB cursor", "/api/models?author=pub&cursor=" + cursor(math.MaxInt32-1) + "&limit=2", public, nil, []call{{filter("pub", math.MaxInt32, 1, "", nil), models[:1], math.MaxInt32 + 1, nil}}, false, http.StatusOK, "pub/alpha", "", ""},
		{"unaligned final DB page", "/api/models?author=pub&cursor=" + cursor(math.MaxInt32-2) + "&limit=2", public, nil, []call{{filter("pub", math.MaxInt32/2, 2, "", nil), models[:2], math.MaxInt32, nil}, {filter("pub", math.MaxInt32/2+1, 2, "", nil), models[2:], math.MaxInt32, nil}}, false, http.StatusOK, "pub/beta,pub/beta-2", "", ""},
		{"list query error", "/api/models?author=pub", public, nil, []call{{filter("pub", 1, 100, "", nil), nil, 0, dbErr}}, false, http.StatusInternalServerError, "", `{"error":"database offline"}`, ""},
		{"datasets", "/api/datasets?author=pub", public, nil, []call{{filter("pub", 1, 100, "", nil), nil, 1, nil}}, true, http.StatusOK, "pub/data", dataItem, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			backend, err := New(t.Context(), testConfig(t.TempDir()))
			if err != nil {
				t.Fatal(err)
			}
			backend.sessionRepo = &handlerSessionRepo{manager: scs.New()}
			backend.permissionHookFunc = func(context.Context, permission.Operation, string, permission.Context) (bool, error) {
				return true, nil
			}
			if _, err := repository.Init(t.Context(), backend.Storage().RepositoriesFS(), "/pub/ghost.git", "main"); err != nil {
				t.Fatal(err)
			}
			ctrl := gomock.NewController(t)
			projectRepo := projectmocks.NewMockIProjectRepo(ctrl)
			if test.project != nil || test.projectErr != nil {
				author, _ := url.Parse(test.path)
				projectRepo.EXPECT().GetProjectByName(gomock.Any(), author.Query().Get("author")).Return(test.project, test.projectErr)
			}
			backend.projectRepo = projectRepo
			modelService := modelmocks.NewMockIModelService(ctrl)
			datasetService := &handlerDatasetService{}
			for _, c := range test.calls {
				if test.datasets {
					datasetService.lists = append(datasetService.lists, c.filter)
					datasetService.rows, datasetService.total, datasetService.err = []*dataset.Dataset{{ProjectName: "pub", Name: "data"}}, c.total, c.err
					continue
				}
				modelService.EXPECT().ListModels(gomock.Any(), c.filter).DoAndReturn(func(_ context.Context, got *model.Filter) ([]*model.Model, int64, error) {
					offset := (got.Page - 1) * got.PageSize
					if wideOffset := (int64(got.Page) - 1) * int64(got.PageSize); offset < 0 || int64(offset) != wideOffset {
						t.Errorf("DB offset overflow for filter %+v: got %d, want %d", got, offset, wideOffset)
					}
					return c.rows, c.total, c.err
				})
			}
			backend.modelService, backend.datasetService = modelService, datasetService
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			response := httptest.NewRecorder()
			backend.Handler(http.NotFoundHandler()).ServeHTTP(response, request)
			if response.Code != test.status || response.Header().Get("Link") != test.link {
				t.Fatalf("status = %d, Link = %q, body = %s; want %d, %q", response.Code, response.Header().Get("Link"), response.Body.String(), test.status, test.link)
			}
			if got := strings.TrimSpace(response.Body.String()); test.body != "" && got != test.body {
				t.Errorf("body = %s, want %s", got, test.body)
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
			if len(datasetService.lists) != 0 {
				t.Errorf("dataset filters not consumed: %+v", datasetService.lists)
			}
		})
	}
}

// handlerDatasetService fails ListDatasets unless its filter is the next expected one in lists.
type handlerDatasetService struct {
	dataset.IDatasetService
	lists []*model.Filter
	rows  []*dataset.Dataset
	total int64
	err   error
	calls []string
}

func (service *handlerDatasetService) ListDatasets(_ context.Context, filter *model.Filter) ([]*dataset.Dataset, int64, error) {
	if len(service.lists) == 0 || !reflect.DeepEqual(service.lists[0], filter) {
		return nil, 0, fmt.Errorf("unexpected ListDatasets(%+v), want %+v", filter, service.lists)
	}
	service.lists = service.lists[1:]
	return service.rows, service.total, service.err
}

func (service *handlerDatasetService) CreateDataset(_ context.Context, project, name string) (*dataset.Dataset, error) {
	service.calls = append(service.calls, "create "+project+"/"+name)
	return &dataset.Dataset{ProjectName: project, Name: name}, service.err
}

func (service *handlerDatasetService) DeleteDataset(_ context.Context, project, name string) error {
	service.calls = append(service.calls, "delete "+project+"/"+name)
	return service.err
}

func TestHandlerRepoCRUD(t *testing.T) {
	backend, _ := New(t.Context(), testConfig(t.TempDir()))
	backend.sessionRepo = &handlerSessionRepo{manager: scs.New()}
	backend.permissionHookFunc = func(context.Context, permission.Operation, string, permission.Context) (bool, error) {
		return true, nil
	}
	modelService := modelmocks.NewMockIModelService(gomock.NewController(t))
	datasetService := &handlerDatasetService{}
	backend.modelService, backend.datasetService = modelService, datasetService
	repositories := backend.Storage().RepositoriesFS()
	if _, err := repository.Init(t.Context(), repositories, "/proj/ghost.git", "main"); err != nil {
		t.Fatal(err)
	}
	handler := backend.Handler(http.NotFoundHandler())
	dbErr := errors.New("database offline")

	const created = `{"url":"http://example.com/proj/repo"}`
	for _, step := range []struct {
		name, method, path, body string
		expect                   func()
		datasetErr               error
		status                   int
		response                 string
		datasetCalls             string
	}{
		{"create", http.MethodPost, "/api/repos/create", `{"type":"model","name":"repo","organization":"proj"}`,
			func() { modelService.EXPECT().CreateModel(gomock.Any(), "proj", "repo").Return(&model.Model{}, nil) }, nil, http.StatusOK, created, ""},
		{"create existing", http.MethodPost, "/api/repos/create", `{"type":"model","name":"repo","organization":"proj"}`,
			func() {
				modelService.EXPECT().CreateModel(gomock.Any(), "proj", "repo").Return(nil, errors.New("model already exists"))
			}, nil, http.StatusOK, created, ""},
		{"create in missing project", http.MethodPost, "/api/repos/create", `{"type":"model","name":"repo","organization":"nope"}`,
			func() {
				modelService.EXPECT().CreateModel(gomock.Any(), "nope", "repo").Return(nil, errors.New("project not found: nope"))
			}, nil, http.StatusNotFound, "", ""},
		{"create query error", http.MethodPost, "/api/repos/create", `{"type":"model","name":"repo","organization":"proj"}`,
			func() { modelService.EXPECT().CreateModel(gomock.Any(), "proj", "repo").Return(nil, dbErr) }, nil, http.StatusInternalServerError, `{"error":"database offline"}`, ""},
		{"create dataset", http.MethodPost, "/api/repos/create", `{"type":"dataset","name":"data","organization":"proj"}`,
			nil, nil, http.StatusOK, `{"url":"http://example.com/datasets/proj/data"}`, "create proj/data"},
		{"create existing dataset", http.MethodPost, "/api/repos/create", `{"type":"dataset","name":"data","organization":"proj"}`,
			nil, errors.New("dataset already exists"), http.StatusOK, `{"url":"http://example.com/datasets/proj/data"}`, "create proj/data"},
		{"create space", http.MethodPost, "/api/repos/create", `{"type":"space","name":"app","organization":"proj"}`,
			nil, nil, http.StatusInternalServerError, `{"error":"unsupported repository \"spaces/proj/app\""}`, ""},
		{"create kernel", http.MethodPost, "/api/repos/create", `{"type":"kernel","name":"k","organization":"proj"}`,
			nil, nil, http.StatusInternalServerError, `{"error":"unsupported repository \"kernels/proj/k\""}`, ""},
		{"delete", http.MethodDelete, "/api/repos/delete", `{"type":"model","name":"repo","organization":"proj"}`,
			func() { modelService.EXPECT().DeleteModel(gomock.Any(), "proj", "repo").Return(nil) }, nil, http.StatusOK, "", ""},
		{"delete missing", http.MethodDelete, "/api/repos/delete", `{"type":"model","name":"ghost","organization":"proj"}`,
			func() {
				modelService.EXPECT().DeleteModel(gomock.Any(), "proj", "ghost").Return(errors.New("failed to get model: record not found"))
			}, nil, http.StatusNotFound, "", ""},
		{"delete without repository", http.MethodDelete, "/api/repos/delete", `{"type":"model","name":"gone","organization":"proj"}`,
			func() {
				modelService.EXPECT().DeleteModel(gomock.Any(), "proj", "gone").Return(errors.New("repository does not exist at /proj/gone.git"))
			}, nil, http.StatusNotFound, "", ""},
		{"delete query error", http.MethodDelete, "/api/repos/delete", `{"type":"model","name":"repo","organization":"proj"}`,
			func() { modelService.EXPECT().DeleteModel(gomock.Any(), "proj", "repo").Return(dbErr) }, nil, http.StatusInternalServerError, `{"error":"database offline"}`, ""},
		{"delete dataset", http.MethodDelete, "/api/repos/delete", `{"type":"dataset","name":"data","organization":"proj"}`,
			nil, nil, http.StatusOK, "", "delete proj/data"},
		{"delete space", http.MethodDelete, "/api/repos/delete", `{"type":"space","name":"app","organization":"proj"}`,
			nil, nil, http.StatusInternalServerError, `{"error":"unsupported repository \"spaces/proj/app\""}`, ""},
		{"settings disabled", http.MethodPut, "/api/models/proj/repo/settings", `{"private":true}`, nil, nil, http.StatusNotFound, "", ""},
		{"move disabled", http.MethodPost, "/api/repos/move", `{"fromRepo":"proj/repo","toRepo":"proj/moved","type":"model"}`, nil, nil, http.StatusNotFound, "", ""},
	} {
		if step.expect != nil {
			step.expect()
		}
		datasetService.calls, datasetService.err = nil, step.datasetErr
		request := httptest.NewRequest(step.method, step.path, strings.NewReader(step.body))
		request = request.WithContext(authenticate.WithIdentity(request.Context(), middleware.Principal{Identity: user.NewUserIdentity(1, "alice")}))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != step.status || (step.response != "" && strings.TrimSpace(response.Body.String()) != step.response) {
			t.Fatalf("%s: status = %d, body = %s; want %d, %s", step.name, response.Code, response.Body.String(), step.status, step.response)
		}
		if got := strings.Join(datasetService.calls, ","); got != step.datasetCalls {
			t.Fatalf("%s: dataset calls = %q, want %q", step.name, got, step.datasetCalls)
		}
	}
	for path, want := range map[string]bool{"/proj/repo.git": false, "/datasets/proj/data.git": false, "/proj/ghost.git": true} {
		if got := repository.IsRepository(repositories, path); got != want {
			t.Errorf("IsRepository(%s) = %t, want %t", path, got, want)
		}
	}
}

func TestHandlerSignedTokenFromSSHIdentity(t *testing.T) {
	backend, err := New(t.Context(), testConfig(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	backend.sessionRepo = &handlerSessionRepo{manager: scs.New()}
	backend.permissionHookFunc = middleware.NewRepoEnforcer(handlerAuthzService{})
	if _, err := repository.Init(t.Context(), backend.Storage().RepositoriesFS(), "/priv/model.git", "main"); err != nil {
		t.Fatal(err)
	}
	handler := backend.Handler(http.NotFoundHandler())
	const batch = "/priv/model.git/info/lfs/objects/batch"
	sign := func(t *testing.T, ctx context.Context, subject string) string {
		t.Helper()
		token, err := backend.auth.tokenSignValidator.Sign(ctx, http.MethodPost, "http://host"+batch, subject, time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		return token
	}
	// git-lfs-authenticate signs with the SSH session's built-in identity, whose name is the encoded user.
	sshToken := func(t *testing.T, identity auth.Identity) string {
		encoded := encodedIdentity(t, identity)
		return sign(t, authenticate.WithIdentity(context.Background(), authenticate.NewIdentity(encoded, "")), encoded)
	}
	legacy, err := authenticate.NewTokenSignValidator([]byte("test-secret")).Sign(context.Background(), http.MethodPost, "http://host"+batch, "alice", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		token  string
		status int
	}{
		{"member", sshToken(t, user.NewUserIdentity(1, "alice")), http.StatusOK},
		{"other user", sshToken(t, user.NewUserIdentity(2, "bob")), http.StatusForbidden},
		{"plain subject", legacy, http.StatusUnauthorized},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := `{"operation":"download","objects":[{"oid":"` + strings.Repeat("ab", 32) + `","size":1}]}`
			request := httptest.NewRequest(http.MethodPost, batch, strings.NewReader(body))
			request.Header.Set("Authorization", "Bearer "+test.token)
			request.Header.Set("Accept", "application/vnd.git-lfs+json")
			request.Header.Set("Content-Type", "application/vnd.git-lfs+json")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.status, response.Body.String())
			}
			if test.status == http.StatusOK && !strings.Contains(response.Body.String(), `"transfer":"basic"`) {
				t.Fatalf("body = %s, want a batch response", response.Body.String())
			}
		})
	}
}

func TestHandlerServesColdLFSResolveFromSharedMirror(t *testing.T) {
	for _, firstMethod := range []string{http.MethodHead, http.MethodGet} {
		t.Run(firstMethod, func(t *testing.T) { testColdLFSResolve(t, firstMethod) })
	}
}

// testColdLFSResolve runs the cold first-request flow against a source serving
// its resolve endpoint under a base path, redirecting to a CDN on another port.
func testColdLFSResolve(t *testing.T, firstMethod string) {
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	data := make([]byte, 96<<10)
	_, _ = rand.NewChaCha8([32]byte{1}).Read(data)
	sum := sha256.Sum256(data)
	oid := hex.EncodeToString(sum[:])
	size := fmt.Sprint(len(data))
	const head, first, remoteRepo, localRepo, base = 48 << 10, 40 << 10, "remote-org/repo", "local/repo", "/hub"

	remote, err := gitstorage.NewStorage(gitstorage.WithRootDir(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	src, err := repository.Init(ctx, remote.RepositoriesFS(), repository.ResolvePath(remoteRepo), "main")
	if err != nil {
		t.Fatal(err)
	}
	commit, err := src.CreateCommit(ctx, "main", "weights", "Test", "test@test.com", []repository.CommitOperation{
		{Type: repository.CommitOperationAdd, Path: "README.md", Content: []byte("hello")},
		{Type: repository.CommitOperationAdd, Path: "weights/model.bin", Content: fmt.Appendf(nil, "version https://git-lfs.github.com/spec/v1\noid sha256:%s\nsize %d\n", oid, len(data))},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	var (
		mu                  sync.Mutex
		resolves, downloads int
		requests            int
		release             = make(chan struct{})
		releaseOnce         sync.Once
		upstream, cdn       *httptest.Server
	)
	counts := func() (int, int, int) {
		mu.Lock()
		defer mu.Unlock()
		return requests, resolves, downloads
	}
	serve := func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		downloads++
		mu.Unlock()
		w.Header().Set("Content-Length", size)
		_, _ = w.Write(data[:head])
		w.(http.Flusher).Flush()
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		_, _ = w.Write(data[head:])
	}
	gitHandler := http.StripPrefix(base, backendhttp.NewHandler(backendhttp.WithStorage(remote)))
	upstream = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests++
		mu.Unlock()
		switch r.URL.Path {
		case base + "/" + remoteRepo + "/resolve/" + commit + "/weights/model.bin":
			mu.Lock()
			resolves++
			mu.Unlock()
			if r.Header.Get("Authorization") != "Bearer upstream-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, cdn.URL+"/cdn/"+oid, http.StatusFound)
		default:
			gitHandler.ServeHTTP(w, r)
		}
	}))
	t.Cleanup(upstream.Close)
	// Another port is another origin: the registry credential must not follow the redirect here.
	cdn = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cdn/"+oid || r.Header.Get("Authorization") != "" {
			t.Errorf("cdn %s %s with Authorization %q, want the object path without credentials", r.Method, r.URL.Path, r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Method == http.MethodHead {
			w.Header().Set("Content-Length", size)
			return
		}
		serve(w, r)
	}))
	t.Cleanup(cdn.Close)

	b, err := New(ctx, &config.Config{DataDir: t.TempDir(), APIServer: &config.APIServerConfig{TokenSigningSecret: "test-secret"}})
	if err != nil {
		t.Fatal(err)
	}
	b.sessionRepo = &handlerSessionRepo{manager: scs.New()}
	b.auth.tokenValidator = authenticate.NewSimpleTokenValidator(encodedIdentity(t, user.NewUserIdentity(42, "alice")), "alice-token")
	b.permissionHookFunc = func(ctx context.Context, _ permission.Operation, repoName string, _ permission.Context) (bool, error) {
		return repoName == localRepo && !authenticate.IsAnonymous(authenticate.IdentityFrom(ctx)), nil
	}
	var pullOnce sync.Once
	var pullErr error
	ctrl := gomock.NewController(t)
	modelService := modelmocks.NewMockIModelService(ctrl)
	modelService.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "local", "repo").DoAndReturn(func(ctx context.Context, _, _ string) error {
		pullOnce.Do(func() {
			pullErr = b.Mirror().PullFromRemote(ctx, repository.ResolvePath(localRepo), localRepo, &mirror.PullOptions{
				SourceURL: upstream.URL + base + "/" + remoteRepo,
				UserInfo:  url.UserPassword("upstream", "upstream-token"),
				Output:    io.Discard,
			})
		})
		return pullErr
	}).AnyTimes()
	b.modelService = modelService
	registryID := 1
	reg := &registry.Registry{ID: registryID, URL: upstream.URL + base}
	reg.SetCredential(registry.NewBasicCredential("upstream", "upstream-token"))
	projectRepo := projectmocks.NewMockIProjectRepo(ctrl)
	projectRepo.EXPECT().GetProjectByName(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, name string) (*project.Project, error) {
		if name != "local" {
			return nil, gorm.ErrRecordNotFound
		}
		return &project.Project{Name: name, Organization: "remote-org", RegistryID: &registryID}, nil
	}).AnyTimes()
	registryRepo := registrymocks.NewMockIRegistryRepo(ctrl)
	registryRepo.EXPECT().GetRegistry(gomock.Any(), registryID).Return(reg, nil).AnyTimes()
	b.projectRepo, b.registryRepo = projectRepo, registryRepo
	server := httptest.NewServer(b.Handler(http.NotFoundHandler()))
	t.Cleanup(server.Close)
	t.Cleanup(b.Mirror().Wait)
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })

	do := func(ctx context.Context, method, path, token string, header http.Header, follow bool) *http.Response {
		t.Helper()
		request, err := http.NewRequestWithContext(ctx, method, server.URL+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		if token != "" {
			request.Header.Set("Authorization", "Bearer "+token)
		}
		for key, values := range header {
			request.Header[key] = values
		}
		client := &http.Client{}
		if !follow {
			client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		}
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = response.Body.Close() })
		return response
	}
	const file = "/" + localRepo + "/resolve/main/weights/model.bin"

	response := do(ctx, http.MethodHead, file, "", nil, true)
	if total, _, _ := counts(); response.StatusCode != http.StatusForbidden || total != 0 {
		t.Fatalf("anonymous HEAD: status = %d, upstream requests = %d; want 403 and none", response.StatusCode, total)
	}

	response = do(ctx, firstMethod, file, "alice-token", nil, true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("cold %s status = %d, want 200 (pull error: %v)", firstMethod, response.StatusCode, pullErr)
	}
	for key, want := range map[string]string{"X-Repo-Commit": commit, "ETag": `"` + oid + `"`, "X-Linked-Size": size, "Content-Length": size} {
		if got := response.Header.Get(key); got != want {
			t.Errorf("cold %s %s = %q, want %q", firstMethod, key, got, want)
		}
	}
	if got := response.Header.Get("Content-Disposition"); !strings.Contains(got, "model.bin") {
		t.Errorf("cold %s Content-Disposition = %q, want the file name", firstMethod, got)
	}

	if firstMethod == http.MethodHead {
		response = do(ctx, http.MethodGet, file, "alice-token", nil, true)
	}
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Length") != size {
		t.Fatalf("cold GET status = %d, Content-Length = %q; want 200 and %s", response.StatusCode, response.Header.Get("Content-Length"), size)
	}
	got := make([]byte, len(data))
	if _, err := io.ReadFull(response.Body, got[:first]); err != nil {
		t.Fatalf("cold GET first bytes before release: %v", err)
	}
	if b.Mirror().HasObject(ctx, oid) {
		t.Fatal("object landed in the shared storage before the upstream released the body")
	}

	ranged := do(ctx, http.MethodGet, file, "alice-token", http.Header{"Range": {"bytes=1024-2047"}}, true)
	part, err := io.ReadAll(ranged.Body)
	if err != nil || ranged.StatusCode != http.StatusPartialContent || ranged.Header.Get("Content-Range") != "bytes 1024-2047/"+size || !bytes.Equal(part, data[1024:2048]) {
		t.Fatalf("mid-ingest range: status = %d, Content-Range = %q, %d bytes, err = %v", ranged.StatusCode, ranged.Header.Get("Content-Range"), len(part), err)
	}

	// A downstream client giving up must not cancel the shared ingest.
	abandonCtx, abandon := context.WithCancel(ctx)
	abandoned := do(abandonCtx, http.MethodGet, file, "alice-token", nil, true)
	if _, err := io.ReadFull(abandoned.Body, make([]byte, 1)); err != nil {
		t.Fatalf("abandoned GET first byte: %v", err)
	}
	abandon()

	releaseOnce.Do(func() { close(release) })
	if _, err := io.ReadFull(response.Body, got[first:]); err != nil {
		t.Fatalf("cold GET tail: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatal("cold GET body differs from the upstream payload")
	}
	b.Mirror().Wait()
	if !b.Mirror().HasObject(ctx, oid) {
		t.Fatal("object was not published to the shared storage after the ingest")
	}

	response = do(ctx, http.MethodGet, file, "alice-token", nil, false)
	location := response.Header.Get("Location")
	if response.StatusCode != http.StatusFound || !strings.HasSuffix(location, "/xet-bridge/"+oid) || response.Header.Get("X-Xet-Hash") == "" {
		t.Fatalf("warm GET status = %d, Location = %q, X-Xet-Hash = %q; want 302 to the bridge with the xet hash", response.StatusCode, location, response.Header.Get("X-Xet-Hash"))
	}
	bridged := do(ctx, http.MethodGet, strings.TrimPrefix(location, server.URL), "", nil, true)
	body, err := io.ReadAll(bridged.Body)
	if err != nil || bridged.StatusCode != http.StatusOK || !bytes.Equal(body, data) {
		t.Fatalf("bridge GET status = %d, %d bytes, err = %v; want the full payload", bridged.StatusCode, len(body), err)
	}

	// Denied or unresolvable downstream requests never reach the upstream.
	before, _, _ := counts()
	for _, test := range []struct {
		name, method, path, token string
		status                    int
	}{
		{"wrong token", http.MethodGet, file, "not-the-token", http.StatusUnauthorized},
		{"anonymous GET", http.MethodGet, file, "", http.StatusForbidden},
		{"other repository", http.MethodGet, "/other/repo/resolve/main/weights/model.bin", "alice-token", http.StatusForbidden},
		{"missing file", http.MethodGet, "/" + localRepo + "/resolve/main/nope.bin", "alice-token", http.StatusNotFound},
		{"regular file", http.MethodGet, "/" + localRepo + "/resolve/main/README.md", "alice-token", http.StatusOK},
	} {
		response := do(ctx, test.method, test.path, test.token, nil, false)
		if response.StatusCode != test.status {
			t.Errorf("%s: status = %d, want %d", test.name, response.StatusCode, test.status)
		}
	}
	// One HEAD probe and one GET fetch reach the source's resolve endpoint.
	if total, nr, nd := counts(); total != before || nr != 2 || nd != 1 {
		t.Fatalf("requests = %d, resolves = %d, downloads = %d; want %d, 2, 1", total, nr, nd, before)
	}
}

// HF API writes into a pull-mirrored repository are refused, as main's
// CreateModelCommit did; git push is not gated here and keeps main's behaviour.
func TestHandlerRefusesHFWritesToProxiedRepository(t *testing.T) {
	ctx := t.Context()
	b, err := New(ctx, testConfig(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	b.sessionRepo = &handlerSessionRepo{manager: scs.New()}
	b.auth.tokenValidator = authenticate.NewSimpleTokenValidator(encodedIdentity(t, user.NewUserIdentity(42, "alice")), "alice-token")
	b.permissionHookFunc = func(context.Context, permission.Operation, string, permission.Context) (bool, error) {
		return true, nil
	}
	registryID := 1
	reg := &registry.Registry{ID: registryID, URL: "https://hub.example"}
	reg.SetCredential(registry.NewBasicCredential("upstream", "upstream-token"))
	ctrl := gomock.NewController(t)
	projectRepo := projectmocks.NewMockIProjectRepo(ctrl)
	projectRepo.EXPECT().GetProjectByName(gomock.Any(), "local").Return(&project.Project{Name: "local", Organization: "remote-org", RegistryID: &registryID}, nil).AnyTimes()
	projectRepo.EXPECT().GetProjectByName(gomock.Any(), "plain").Return(&project.Project{Name: "plain"}, nil).AnyTimes()
	registryRepo := registrymocks.NewMockIRegistryRepo(ctrl)
	registryRepo.EXPECT().GetRegistry(gomock.Any(), registryID).Return(reg, nil).AnyTimes()
	modelService := modelmocks.NewMockIModelService(ctrl)
	b.projectRepo, b.registryRepo, b.modelService = projectRepo, registryRepo, modelService
	if _, err := repository.Init(ctx, b.Storage().RepositoriesFS(), repository.ResolvePath("plain/repo"), "main"); err != nil {
		t.Fatal(err)
	}
	handler := b.Handler(http.NotFoundHandler())

	do := func(method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer alice-token")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}
	const commit = "{\"key\":\"header\",\"value\":{\"summary\":\"upload\"}}\n" +
		"{\"key\":\"file\",\"value\":{\"path\":\"README.md\",\"content\":\"# hello\\n\"}}\n"

	// No EnsureModel expectation: a proxied write must be refused before any hook runs.
	if response := do(http.MethodPost, "/api/models/local/repo/commit/main", commit); response.Code != http.StatusForbidden {
		t.Fatalf("proxied commit: status = %d, body = %s; want 403", response.Code, response.Body.String())
	}
	if response := do(http.MethodPost, "/api/models/local/repo/preupload/main", `{"files":[{"path":"README.md","size":8}]}`); response.Code != http.StatusForbidden {
		t.Fatalf("proxied preupload: status = %d, body = %s; want 403", response.Code, response.Body.String())
	}

	modelService.EXPECT().EnsureModel(gomock.Any(), "local", "repo").Return(&model.Model{}, nil)
	if response := do(http.MethodGet, "/local/repo.git/info/refs?service=git-receive-pack", ""); response.Code == http.StatusForbidden {
		t.Fatalf("git push discovery into the proxied repository was refused; want main's behaviour (allowed: 404 here as no repository exists on disk)")
	}

	modelService.EXPECT().EnsureModel(gomock.Any(), "plain", "repo").Return(&model.Model{}, nil).MinTimes(1)
	modelService.EXPECT().SyncMetadata(gomock.Any(), "plain", "repo").Return(nil)
	if response := do(http.MethodPost, "/api/models/plain/repo/commit/main", commit); response.Code != http.StatusOK {
		t.Fatalf("plain commit: status = %d, body = %s; want 200", response.Code, response.Body.String())
	}
}

// Nested model names ("proj/a/b") parse but are unaddressable by the REST API and UI, so the hooks must not create or sync them.
func TestHandlerNestedModelNameIsNotAddressable(t *testing.T) {
	ctx := t.Context()
	b, err := New(ctx, testConfig(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	b.sessionRepo = &handlerSessionRepo{manager: scs.New()}
	b.permissionHookFunc = func(context.Context, permission.Operation, string, permission.Context) (bool, error) {
		return true, nil
	}
	// No EnsureModel or CheckOrSyncFromRemote expectation: any hook call fails the test.
	b.modelService = modelmocks.NewMockIModelService(gomock.NewController(t))
	handler := b.Handler(http.NotFoundHandler())
	for _, test := range []struct{ name, path string }{
		{"push discovery", "/proj/a/b.git/info/refs?service=git-receive-pack"},
		{"clone discovery", "/proj/a/b.git/info/refs?service=git-upload-pack"},
		{"resolve", "/proj/a/b/resolve/main/README.md"},
	} {
		request := httptest.NewRequest(http.MethodGet, test.path, nil)
		request = request.WithContext(authenticate.WithIdentity(request.Context(), middleware.Principal{Identity: user.NewUserIdentity(1, "alice")}))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Errorf("%s: status = %d, body = %s; want 404", test.name, response.Code, response.Body.String())
		}
	}
	if ok, err := b.preReceiveHook(ctx, "proj/a/b", nil); ok || err != nil {
		t.Errorf("preReceiveHook(proj/a/b) = (%t, %v), want (false, nil)", ok, err)
	}
	// hfd's SSH command path compares the sentinel with ==, so it must not be wrapped.
	for _, write := range []bool{false, true} {
		if err := b.preOpenHook(ctx, "proj/a/b", write); err != repository.ErrRepositoryNotExists {
			t.Errorf("preOpenHook(proj/a/b, write=%t) = %v, want the bare repository.ErrRepositoryNotExists", write, err)
		}
	}
}

// The hfd mirror's SourceFunc answers with the repository's registry source
// URL carrying the registry's Basic credential as userinfo.
func TestBackendMirrorSourceURL(t *testing.T) {
	ctx := t.Context()
	b, err := New(ctx, &config.Config{DataDir: t.TempDir(), APIServer: &config.APIServerConfig{TokenSigningSecret: "test-secret"}})
	if err != nil {
		t.Fatal(err)
	}
	registryID := 7
	dbErr := errors.New("database offline")
	ctrl := gomock.NewController(t)
	projectRepo := projectmocks.NewMockIProjectRepo(ctrl)
	registryRepo := registrymocks.NewMockIRegistryRepo(ctrl)
	b.projectRepo, b.registryRepo = projectRepo, registryRepo
	proxied := func() *gomock.Call {
		return projectRepo.EXPECT().GetProjectByName(gomock.Any(), "local").Return(&project.Project{Name: "local", Organization: "remote-org", RegistryID: &registryID}, nil)
	}
	source := func(rawURL string) *gomock.Call {
		r := &registry.Registry{ID: registryID, URL: rawURL}
		r.SetCredential(registry.NewBasicCredential("upstream", "upstream-token"))
		return registryRepo.EXPECT().GetRegistry(gomock.Any(), registryID).Return(r, nil)
	}
	gomock.InOrder(
		proxied(), source("https://first.example/hub/"),
		proxied(), source("http://second.example:8080"),
		proxied(), registryRepo.EXPECT().GetRegistry(gomock.Any(), registryID).Return(&registry.Registry{ID: registryID, URL: "https://public.example"}, nil),
		projectRepo.EXPECT().GetProjectByName(gomock.Any(), "plain").Return(&project.Project{Name: "plain"}, nil),
		projectRepo.EXPECT().GetProjectByName(gomock.Any(), "missing").Return(nil, gorm.ErrRecordNotFound),
		projectRepo.EXPECT().GetProjectByName(gomock.Any(), "local").Return(nil, dbErr),
		proxied(), registryRepo.EXPECT().GetRegistry(gomock.Any(), registryID).Return(nil, dbErr),
	)
	for _, test := range []struct {
		name, repo, url string
		ok              bool
		err             error
	}{
		{"registry with a base path", "local/repo", "https://upstream:upstream-token@first.example/hub/remote-org/repo", true, nil},
		{"registry with a port and no trailing slash", "local/repo", "http://upstream:upstream-token@second.example:8080/remote-org/repo", true, nil},
		{"registry without a credential", "local/repo", "https://public.example/remote-org/repo", true, nil},
		{"non-proxy project", "plain/repo", "", false, nil},
		{"missing project", "missing/repo", "", false, nil},
		{"project query error", "local/repo", "", false, dbErr},
		{"registry query error", "local/repo", "", false, dbErr},
		{"dataset", "datasets/local/repo", "", false, nil}, // no expectation is left: a DB call would fail the mocks
	} {
		got, ok, err := b.mirrorSource(ctx, test.repo)
		if got != test.url || ok != test.ok || !errors.Is(err, test.err) {
			t.Errorf("%s: mirrorSource = %q, %t, %v; want %q, %t, %v", test.name, got, ok, err, test.url, test.ok, test.err)
		}
	}
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
