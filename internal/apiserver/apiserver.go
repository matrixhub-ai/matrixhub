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

package apiserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/matrixhub-ai/hfd/pkg/authenticate"
	backendssh "github.com/matrixhub-ai/hfd/pkg/backend/ssh"
	"github.com/matrixhub-ai/hfd/pkg/lfs"
	"github.com/matrixhub-ai/hfd/pkg/mirror"
	"github.com/matrixhub-ai/hfd/pkg/permission"
	"github.com/matrixhub-ai/hfd/pkg/receive"
	hfdssh "github.com/matrixhub-ai/hfd/pkg/ssh"
	gitstorage "github.com/matrixhub-ai/hfd/pkg/storage"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/matrixhub-ai/matrixhub/internal/apiserver/handler"
	backendhf "github.com/matrixhub-ai/matrixhub/internal/apiserver/handler/hf"
	backendhttp "github.com/matrixhub-ai/matrixhub/internal/apiserver/handler/http"
	backendlfs "github.com/matrixhub-ai/matrixhub/internal/apiserver/handler/lfs"
	"github.com/matrixhub-ai/matrixhub/internal/apiserver/middleware"
	"github.com/matrixhub-ai/matrixhub/internal/domain/authz"
	"github.com/matrixhub-ai/matrixhub/internal/domain/dataset"
	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registrydiscovery"
	"github.com/matrixhub-ai/matrixhub/internal/domain/syncjob"
	"github.com/matrixhub-ai/matrixhub/internal/domain/syncpolicy"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/config"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
	"github.com/matrixhub-ai/matrixhub/internal/jobserver"
	"github.com/matrixhub-ai/matrixhub/internal/jobserver/canceller"
	"github.com/matrixhub-ai/matrixhub/internal/jobserver/logstore"
	"github.com/matrixhub-ai/matrixhub/internal/repo"
	hfdiscovery "github.com/matrixhub-ai/matrixhub/internal/repo/registrydiscovery/hf"
	mhdiscovery "github.com/matrixhub-ai/matrixhub/internal/repo/registrydiscovery/matrixhub"
)

const maxGrpcMsgSize = 100 * 1024 * 1024

type APIServer struct {
	config     *config.Config
	debug      bool
	cmux       cmux.CMux
	httpServer *http.Server
	sshServer  *backendssh.Server
	engine     *gin.Engine
	gatewayMux *runtime.ServeMux
	grpcServer *grpc.Server
	port       int

	gitHooks   gitHooks
	gitAuth    gitAuth
	gitStorage gitStorage

	repos    *repo.Repos
	services *Services
	handlers []handler.IHandler

	jobServer *jobserver.JobServer
	jobCancel context.CancelFunc
	jobWait   sync.WaitGroup
}

func NewAPIServer(config *config.Config) *APIServer {
	if config.APIServer == nil {
		panic("apiserver config is nil")
	}

	engine := gin.New()
	engine.Use(
		gin.Recovery(),
	)

	gatewayMux := runtime.NewServeMux(
		runtime.WithForwardResponseOption(middleware.ResponseHeaderLocation),
		runtime.WithOutgoingHeaderMatcher(middleware.HeaderMatcher),
	)

	httpServer := &http.Server{
		Handler:           engine,
		ReadHeaderTimeout: 30 * time.Second,
	}

	server := &APIServer{
		config:     config,
		debug:      config.Debug,
		httpServer: httpServer,
		engine:     engine,
		gatewayMux: gatewayMux,
		port:       config.APIServer.Port,
	}

	server.initGitAuth()
	server.initGitHooks()
	server.initGitStorage()
	server.initSSHBackend()
	server.initHandlersServicesRepos()

	// Register authn + authz middleware (must be after initHandlersServicesRepos)
	streamMiddleware := []grpc.StreamServerInterceptor{
		grpc_recovery.StreamServerInterceptor(),
	}
	unaryMiddleware := []grpc.UnaryServerInterceptor{
		grpc_recovery.UnaryServerInterceptor(),
		middleware.AuthInterceptor(server.repos.Session, server.repos.AccessToken, server.repos.Robot),
		middleware.AuthzInterceptor(server.services.Authz.VerifyPlatformPermission),
	}

	grpcServer := grpc.NewServer(
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			streamMiddleware...,
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			unaryMiddleware...,
		)),
		grpc.MaxSendMsgSize(maxGrpcMsgSize),
		grpc.MaxRecvMsgSize(maxGrpcMsgSize),
	)
	server.grpcServer = grpcServer

	server.httpServer.Handler = server.initBackends(server.httpServer.Handler)
	server.registerRoutersAndHandlers()

	return server
}

type gitHooks struct {
	permissionHookFunc  func(ctx context.Context, op permission.Operation, repoName string, opCtx permission.Context) (bool, error)
	preReceiveHookFunc  func(ctx context.Context, repoName string, updates []receive.RefUpdate) (bool, error)
	postReceiveHookFunc func(ctx context.Context, repoName string, updates []receive.RefUpdate) error
	mirrorSourceFunc    func(ctx context.Context, repoName string) (string, bool, error)
	mirrorRefFilterFunc func(ctx context.Context, repoName string, remoteRefs []string) ([]string, error)
}

func (server *APIServer) initGitHooks() {
	permissionHookFunc := func(ctx context.Context, op permission.Operation, repoName string, opCtx permission.Context) (bool, error) {
		// userInfo, _ := authenticate.GetUserInfo(ctx)
		return true, nil // or return false, nil to deny, or return an error to indicate an error
	}

	preReceiveHookFunc := func(ctx context.Context, repoName string, updates []receive.RefUpdate) (bool, error) {
		// userInfo, _ := authenticate.GetUserInfo(ctx)
		return true, nil // or return false, nil to deny, or return an error to indicate an error
	}

	postReceiveHookFunc := func(ctx context.Context, repoName string, updates []receive.RefUpdate) error {
		return nil
	}

	mirrorSourceFunc := func(ctx context.Context, repoName string) (string, bool, error) {
		// return baseURL + "/" + repoName, true, nil
		return "", false, nil
	}

	mirrorRefFilterFunc := func(ctx context.Context, repoName string, remoteRefs []string) ([]string, error) {
		filteredRefs := []string{}
		for _, ref := range remoteRefs {
			if strings.HasPrefix(ref, "refs/heads/") || strings.HasPrefix(ref, "refs/tags/") {
				filteredRefs = append(filteredRefs, ref)
			}
		}
		return filteredRefs, nil
	}

	server.gitHooks.permissionHookFunc = permissionHookFunc
	server.gitHooks.preReceiveHookFunc = preReceiveHookFunc
	server.gitHooks.postReceiveHookFunc = postReceiveHookFunc
	server.gitHooks.mirrorSourceFunc = mirrorSourceFunc
	server.gitHooks.mirrorRefFilterFunc = mirrorRefFilterFunc
}

type gitAuth struct {
	basicAuthValidator authenticate.BasicAuthValidator
	publicKeyValidator authenticate.PublicKeyValidator
	tokenValidator     authenticate.TokenValidator
	tokenSignValidator authenticate.TokenSignValidator
}

func (server *APIServer) initGitAuth() {
	// TODO: implement real validators that validate the credentials and return the corresponding user info.
	// For now, we just return true for all validators to allow all requests to pass through.
	// This is not secure and should be replaced with real implementations.

	basicAuthValidator := authenticate.BasicAuthValidatorFunc(func(ctx context.Context, username, password string) (user string, next, ok bool, err error) {
		// validate username and password, return true if valid, false if invalid, or an error if there's an error during validation
		return "", true, true, nil
	})

	publicKeyValidator := authenticate.PublicKeyValidatorFunc(func(ctx context.Context, username string, keyType string, marshaledKey []byte) (user string, next, ok bool, err error) {
		// validate the public key for the given username, return true if valid, false if invalid, or an error if there's an error during validation
		return "", true, true, nil
	})

	tokenValidator := authenticate.TokenValidatorFunc(func(ctx context.Context, token string) (user string, next, ok bool, err error) {
		// validate the token, return true if valid, false if invalid, or an error if there's an error during validation
		return "", false, true, nil
	})

	// TODO: Use a proper secret management solution to manage the token signing secret.
	// Generating and validating temporary tokens.
	// Currently only used to provide http lfs download in ssh ports.
	tmpTokenSecret := []byte("secret-xxxxxx")
	tokenSignValidator := authenticate.NewTokenSignValidator(tmpTokenSecret)

	server.gitAuth.basicAuthValidator = basicAuthValidator
	server.gitAuth.publicKeyValidator = publicKeyValidator
	server.gitAuth.tokenValidator = tokenValidator
	server.gitAuth.tokenSignValidator = tokenSignValidator
}

type gitStorage struct {
	storage      *gitstorage.Storage
	lfsStorage   lfs.Storage
	sharedMirror *mirror.Mirror
}

func (server *APIServer) initGitStorage() {
	storage := gitstorage.NewStorage(
		gitstorage.WithRootDir(server.config.DataDir),
	)

	lfsStorage := lfs.NewLocal(storage.LFSDir())

	mirrorSourceFunc := server.gitHooks.mirrorSourceFunc
	mirrorRefFilterFunc := server.gitHooks.mirrorRefFilterFunc
	preReceiveHookFunc := server.gitHooks.preReceiveHookFunc
	postReceiveHookFunc := server.gitHooks.postReceiveHookFunc

	sharedMirror := mirror.NewMirror(
		mirror.WithMirrorSourceFunc(mirrorSourceFunc),
		mirror.WithMirrorRefFilterFunc(mirrorRefFilterFunc),
		mirror.WithPreReceiveHookFunc(preReceiveHookFunc),
		mirror.WithPostReceiveHookFunc(postReceiveHookFunc),
		mirror.WithLFSStorage(lfsStorage),
		mirror.WithXET(false),
		mirror.WithConcurrency(2),
		mirror.WithTTL(time.Minute),
		mirror.WithCacheDir(storage.TmpDir()),
	)

	server.gitStorage.storage = storage
	server.gitStorage.lfsStorage = lfsStorage
	server.gitStorage.sharedMirror = sharedMirror
}

func (server *APIServer) initBackends(handler http.Handler) http.Handler {
	storage := server.gitStorage.storage
	lfsStorage := server.gitStorage.lfsStorage
	sharedMirror := server.gitStorage.sharedMirror
	permissionHookFunc := server.gitHooks.permissionHookFunc
	preReceiveHookFunc := server.gitHooks.preReceiveHookFunc
	postReceiveHookFunc := server.gitHooks.postReceiveHookFunc
	basicAuthValidator := server.gitAuth.basicAuthValidator
	tokenValidator := server.gitAuth.tokenValidator
	tokenSignValidator := server.gitAuth.tokenSignValidator

	handler = backendhf.NewHandler(
		backendhf.WithStorage(storage),
		backendhf.WithNext(handler),
		backendhf.WithMirror(sharedMirror),
		backendhf.WithPermissionHookFunc(permissionHookFunc),
		backendhf.WithPreReceiveHookFunc(preReceiveHookFunc),
		backendhf.WithPostReceiveHookFunc(postReceiveHookFunc),
		backendhf.WithLFSStorage(lfsStorage),
		backendhf.WithMiddlewares(
			middleware.HFAuthnMiddleware(server.repos.AccessToken, server.repos.Session),
			middleware.HFAuthzMiddleware(server.services.Authz),
		),
		backendhf.WithServices(server.services.Model, server.repos.Git, server.services.Authz),
	)

	handler = backendlfs.NewHandler(
		backendlfs.WithStorage(storage),
		backendlfs.WithNext(handler),
		backendlfs.WithMirror(sharedMirror),
		backendlfs.WithPermissionHookFunc(permissionHookFunc),
		backendlfs.WithLFSStorage(lfsStorage),
		backendlfs.WithMirror(sharedMirror),
		backendlfs.WithMiddlewares(
			middleware.GitAuthnMiddleware(server.repos.AccessToken),
			middleware.GitAuthzMiddleware(server.services.Authz),
		),
	)

	handler = backendhttp.NewHandler(
		backendhttp.WithStorage(storage),
		backendhttp.WithNext(handler),
		backendhttp.WithMirror(sharedMirror),
		backendhttp.WithPermissionHookFunc(permissionHookFunc),
		backendhttp.WithPreReceiveHookFunc(preReceiveHookFunc),
		backendhttp.WithPostReceiveHookFunc(postReceiveHookFunc),
		backendhttp.WithMiddlewares(
			middleware.GitAuthnMiddleware(server.repos.AccessToken),
			middleware.GitAuthzMiddleware(server.services.Authz),
		),
	)

	handler = authenticate.AnonymousAuthenticateHandler(handler)
	handler = authenticate.TokenValidatorHandler(tokenValidator, handler)
	handler = authenticate.TokenSignValidatorHandler(tokenSignValidator, handler)
	handler = authenticate.BasicAuthHandler(basicAuthValidator, handler)

	return handler
}

func (server *APIServer) initSSHBackend() {
	if server.config.APIServer.SSHPort == 0 {
		return
	}

	hostKeyPath := server.config.APIServer.SSHHostKeyPath

	storage := server.gitStorage.storage
	sharedMirror := server.gitStorage.sharedMirror
	permissionHookFunc := server.gitHooks.permissionHookFunc
	preReceiveHookFunc := server.gitHooks.preReceiveHookFunc
	postReceiveHookFunc := server.gitHooks.postReceiveHookFunc
	basicAuthValidator := server.gitAuth.basicAuthValidator
	publicKeyValidator := server.gitAuth.publicKeyValidator
	tokenSignValidator := server.gitAuth.tokenSignValidator

	data, _ := os.ReadFile(hostKeyPath)
	hostKeySigner, _ := hfdssh.ParseHostKeyFile(data)
	// TODO: handle error and edge cases for host key file (e.g. file not exist, invalid format, etc.)

	sshOpts := []backendssh.Option{
		backendssh.WithStorage(storage),
		backendssh.WithHostKey(hostKeySigner),
		backendssh.WithPermissionHookFunc(permissionHookFunc),
		backendssh.WithPreReceiveHookFunc(preReceiveHookFunc),
		backendssh.WithPostReceiveHookFunc(postReceiveHookFunc),
		backendssh.WithMirror(sharedMirror),
		backendssh.WithLFSURL(server.config.APIServer.HostURL),
		backendssh.WithBasicAuthValidator(basicAuthValidator),
		backendssh.WithPublicKeyValidator(publicKeyValidator),
		backendssh.WithTokenSignValidator(tokenSignValidator),
	}

	sshServer := backendssh.NewServer(sshOpts...)

	server.sshServer = sshServer
}

type Services struct {
	Model   model.IModelService
	Dataset dataset.IDatasetService
	Authz   authz.IAuthzService
}

func (server *APIServer) initHandlersServicesRepos() {
	// init repos
	repos := repo.NewRepos(server.config,
		server.gitStorage.storage,
		server.gitStorage.sharedMirror,
	)

	// init permission service
	authzService := authz.NewAuthzService(repos.Authz, repos.Project, repos.Robot)

	// init domain services, add if needed
	modelService := model.NewModelService(
		repos.Model,
		repos.Label,
		repos.Git,
		repos.Project,
		repos.Registry,
	)
	datasetService := dataset.NewDatasetService(
		repos.Dataset,
		repos.Label,
		repos.Git,
	)
	userService := user.NewUserService(repos.Session, repos.User)

	logDir := server.config.JobServer.LogDir
	if logDir == "" {
		logDir = filepath.Join(server.config.DataDir, "logs", "jobs")
	}
	logStore := logstore.NewFileLogStore(logDir)
	canc := canceller.NewMemCanceller()

	syncJobService := syncjob.NewSyncJobService(
		repos.SyncJob,
		repos.Registry,
		repos.Project,
		repos.Model,
		repos.Git,
		logStore,
	)

	// init sync policy service
	discoveries := map[string]registrydiscovery.Discovery{
		registrydiscovery.ProviderHuggingFace: hfdiscovery.New(),
		registrydiscovery.ProviderMatrixHub:   mhdiscovery.New(),
	}
	jobGenerator := syncpolicy.NewSyncJobGenerator(repos.Registry, discoveries)
	syncPolicyService := syncpolicy.NewSyncPolicyService(
		repos.SyncPolicy,
		repos.SyncTask,
		syncJobService,
		repos.Project,
		jobGenerator,
	)

	// wire task status reporter from sync policy service to sync job service
	syncJobService.SetOnJobDone(syncPolicyService.ReportTaskStatus)

	if server.config.JobServer != nil && server.config.JobServer.Enabled {
		jc := *server.config.JobServer
		server.jobServer = jobserver.New(&jc, syncPolicyService, syncJobService, logStore, canc)
	}

	server.services = &Services{
		Model:   modelService,
		Dataset: datasetService,
		Authz:   authzService,
	}

	// init handlers
	handlers := []handler.IHandler{
		handler.NewLoginHandler(userService),
		handler.NewRegistryHandler(repos.Registry),
		handler.NewProjectHandler(repos.Project, authzService),
		handler.NewCurrentUserHandler(repos.User, repos.AccessToken, repos.SSHKey),
		handler.NewUserHandler(repos.User, authzService),
		handler.NewDatasetHandler(datasetService),
		handler.NewModelHandler(modelService, authzService),
		handler.NewSyncPolicyHandler(syncPolicyService, syncJobService, repos.Registry, logStore, canc),
		handler.NewRoleHandler(),
		handler.NewRobotHandler(repos.Robot, repos.Project),
	}

	server.repos = repos
	server.handlers = handlers
}

func (server *APIServer) registerRoutersAndHandlers() {
	// healthz endpoint
	server.engine.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "OK") })

	// register routers
	server.engine.Any("/api/v1alpha1/*any", gin.WrapF(server.gatewayMux.ServeHTTP))
	server.engine.Any("/apis/v1alpha1/*any", gin.WrapF(server.gatewayMux.ServeHTTP))

	// serve ui static files if staticDir is configured
	staticDir := server.config.UI.StaticDir
	if staticDir != "" {
		server.engine.Static("/assets", filepath.Join(staticDir, "assets"))
	}

	// SPA fallback - serve index.html for all non-API routes
	server.engine.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if staticDir != "" {
			c.File(filepath.Join(staticDir, "index.html"))
			return
		}
		// If staticDir is not configured, return 404 for non-API routes
		c.JSON(http.StatusNotFound, gin.H{"error": "frontend not configured"})
	})

	options := &handler.ServerOptions{
		GatewayMux: server.gatewayMux,
		GRPCServer: server.grpcServer,
		GRPCAddr:   fmt.Sprintf(":%d", server.port),
		GRPCDialOpt: []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(maxGrpcMsgSize),
				grpc.MaxCallSendMsgSize(maxGrpcMsgSize),
			)},
	}

	for _, h := range server.handlers {
		h.RegisterToServer(options)
	}
}

func (server *APIServer) Start() <-chan error {
	// Create the main listener.
	addr := fmt.Sprintf(":%d", server.port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	server.cmux = cmux.New(l)
	grpcL := server.cmux.Match(cmux.HTTP2())
	httpL := server.cmux.Match(cmux.HTTP1Fast())

	errorCh := make(chan error, 1)
	go func() {
		log.Infof("Internal http server is listening on %s", httpL.Addr().String())
		if err := server.httpServer.Serve(httpL); err != nil {
			errorCh <- err
			if errors.Is(err, http.ErrServerClosed) {
				log.Info("http server closed")
				return
			}
			log.Errorw("run http server failed", "error", err)
		}
	}()

	go func() {
		log.Infof("Internal grpc server is listening on %s", grpcL.Addr().String())
		if err := server.grpcServer.Serve(grpcL); err != nil {
			errorCh <- err
			if errors.Is(err, grpc.ErrServerStopped) {
				log.Info("grpc server closed")
				return
			}
			log.Errorw("run grpc server failed", "error", err)
		}
	}()

	go func() {
		log.Infof("api server is listening on %d", server.port)
		if err := server.cmux.Serve(); err != nil {
			errorCh <- err
			if errors.Is(err, cmux.ErrListenerClosed) {
				log.Info("api server closed")
				return
			}
			log.Errorw("run api server failed", "error", err)
		}
	}()

	if server.config.APIServer.SSHPort != 0 {
		go func() {
			sshAddr := fmt.Sprintf(":%d", server.config.APIServer.SSHPort)
			log.Infof("SSH protocol server is listening on %s", sshAddr)
			if err := server.sshServer.ListenAndServe(context.Background(), sshAddr); err != nil {
				errorCh <- err
				log.Errorw("run SSH protocol server failed", "addr", sshAddr, "error", err)
			}
		}()
	}

	if server.jobServer != nil {
		jsCtx, cancel := context.WithCancel(context.Background())
		server.jobCancel = cancel
		server.jobWait.Add(1)
		go func() {
			defer server.jobWait.Done()
			server.jobServer.Run(jsCtx)
		}()
	}

	return errorCh
}

func (server *APIServer) Shutdown() {
	log.Info("api server shutdown...")

	if server.jobCancel != nil {
		server.jobCancel()
	}
	if server.jobServer != nil {
		grace := server.config.JobServer.ShutdownGrace
		if grace == 0 {
			grace = 30 * time.Second
		}
		server.jobServer.Shutdown(grace)
	}
	server.jobWait.Wait()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	server.cmux.Close()

	if err := server.httpServer.Shutdown(ctx); err != nil {
		log.Error("shutdown error", "error", err)
	}

	server.grpcServer.GracefulStop()

	if err := server.repos.Close(); err != nil {
		log.Error("close db connection error", "error", err)
	}

}
