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
	"time"

	"github.com/gin-gonic/gin"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/matrixhub-ai/hfd/pkg/authenticate"
	backendhf "github.com/matrixhub-ai/hfd/pkg/backend/hf"
	backendhttp "github.com/matrixhub-ai/hfd/pkg/backend/http"
	backendlfs "github.com/matrixhub-ai/hfd/pkg/backend/lfs"
	"github.com/matrixhub-ai/hfd/pkg/lfs"
	"github.com/matrixhub-ai/hfd/pkg/mirror"
	"github.com/matrixhub-ai/hfd/pkg/permission"
	"github.com/matrixhub-ai/hfd/pkg/receive"
	gitstorage "github.com/matrixhub-ai/hfd/pkg/storage"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/matrixhub-ai/matrixhub/internal/apiserver/handler"
	"github.com/matrixhub-ai/matrixhub/internal/apiserver/middleware"
	"github.com/matrixhub-ai/matrixhub/internal/domain/dataset"
	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/config"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
	"github.com/matrixhub-ai/matrixhub/internal/repo"
)

const maxGrpcMsgSize = 100 * 1024 * 1024

type APIServer struct {
	config     *config.Config
	debug      bool
	cmux       cmux.CMux
	httpServer *http.Server
	engine     *gin.Engine
	gatewayMux *runtime.ServeMux
	grpcServer *grpc.Server
	port       int

	gitHooks   gitHooks
	gitStorage gitStorage

	repos    *repo.Repos
	handlers []handler.IHandler
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
		runtime.WithOutgoingHeaderMatcher(func(s string) (string, bool) {
			if s == "Content-Disposition" {
				return s, true
			}
			return fmt.Sprintf("%s%s", runtime.MetadataHeaderPrefix, s), true
		}),
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

	server.initHandlersServicesRepos()

	streamMiddleware := []grpc.StreamServerInterceptor{
		grpc_recovery.StreamServerInterceptor(),
	}
	unaryMiddleware := []grpc.UnaryServerInterceptor{
		grpc_recovery.UnaryServerInterceptor(),
		middleware.AuthInterceptor(server.repos.Session),
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

	server.initGitHooks()
	server.initGitStorage()
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
		// userInfo, _ := authenticate.GetUserInfo(ctx)
		return nil
	}

	mirrorSourceFunc := func(ctx context.Context, repoName string) (string, bool, error) {
		// return baseURL + "/" + repoName, true, nil
		return "", false, nil
	}

	mirrorRefFilterFunc := func(ctx context.Context, repoName string, remoteRefs []string) ([]string, error) {
		return remoteRefs, nil
	}

	server.gitHooks.permissionHookFunc = permissionHookFunc
	server.gitHooks.preReceiveHookFunc = preReceiveHookFunc
	server.gitHooks.postReceiveHookFunc = postReceiveHookFunc
	server.gitHooks.mirrorSourceFunc = mirrorSourceFunc
	server.gitHooks.mirrorRefFilterFunc = mirrorRefFilterFunc
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

	lfsTeeCache := lfs.NewTeeCache(
		lfsStorage,
	)

	mirrorSourceFunc := server.gitHooks.mirrorSourceFunc
	mirrorRefFilterFunc := server.gitHooks.mirrorRefFilterFunc
	preReceiveHookFunc := server.gitHooks.preReceiveHookFunc
	postReceiveHookFunc := server.gitHooks.postReceiveHookFunc

	sharedMirror := mirror.NewMirror(
		mirror.WithMirrorSourceFunc(mirrorSourceFunc),
		mirror.WithMirrorRefFilterFunc(mirrorRefFilterFunc),
		mirror.WithPreReceiveHookFunc(preReceiveHookFunc),
		mirror.WithPostReceiveHookFunc(postReceiveHookFunc),
		mirror.WithLFSCache(lfsTeeCache),
		mirror.WithTTL(time.Hour),
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

	handler = backendhf.NewHandler(
		backendhf.WithStorage(storage),
		backendhf.WithNext(handler),
		backendhf.WithMirror(sharedMirror),
		backendhf.WithPermissionHookFunc(permissionHookFunc),
		backendhf.WithPreReceiveHookFunc(preReceiveHookFunc),
		backendhf.WithPostReceiveHookFunc(postReceiveHookFunc),
		backendhf.WithLFSStorage(lfsStorage),
	)

	handler = backendlfs.NewHandler(
		backendlfs.WithStorage(storage),
		backendlfs.WithNext(handler),
		backendlfs.WithMirror(sharedMirror),
		backendlfs.WithPermissionHookFunc(permissionHookFunc),
		backendlfs.WithLFSStorage(lfsStorage),
		backendlfs.WithMirror(sharedMirror),
	)

	handler = backendhttp.NewHandler(
		backendhttp.WithStorage(storage),
		backendhttp.WithNext(handler),
		backendhttp.WithMirror(sharedMirror),
		backendhttp.WithPermissionHookFunc(permissionHookFunc),
		backendhttp.WithPreReceiveHookFunc(preReceiveHookFunc),
		backendhttp.WithPostReceiveHookFunc(postReceiveHookFunc),
	)

	handler = authenticate.AnonymousAuthenticateHandler(handler)

	return handler
}

func (server *APIServer) initHandlersServicesRepos() {
	// init repos
	repos := repo.NewRepos(server.config,
		server.gitStorage.storage,
		server.gitStorage.sharedMirror,
	)

	// init domain services, add if needed
	modelService := model.NewModelService(
		repos.Model,
		repos.Label,
		repos.Git,
	)
	datasetService := dataset.NewDatasetService(
		repos.Dataset,
		repos.Label,
		repos.Git,
	)
	userService := user.NewUserService(repos.Session, repos.User)

	// init handlers
	handlers := []handler.IHandler{
		handler.NewLoginHandler(userService),
		handler.NewProjectHandler(repos.Project),
		handler.NewUserHandler(repos.User),
		handler.NewCurrentUserHandler(repos.User),
		handler.NewRegistryHandler(),
		handler.NewDatasetHandler(datasetService),
		handler.NewModelHandler(modelService),
	}

	server.repos = repos
	server.handlers = handlers
}

func (server *APIServer) registerRoutersAndHandlers() {
	// healthz endpoint
	server.engine.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "OK") })

	// register routers
	server.engine.Any("/api/v1alpha1/*any", gin.WrapF(server.gatewayMux.ServeHTTP))

	// serve frontend static files
	server.engine.Static("/assets", "/app/assets")

	// SPA fallback - serve index.html for all non-API routes
	server.engine.NoRoute(func(c *gin.Context) {
		// If the request is for an API route that doesn't exist, return 404
		if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		// For all other routes, serve index.html (SPA routing)
		c.File("/app/index.html")
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

	return errorCh
}

func (server *APIServer) Shutdown() {
	log.Info("api server shutdown...")

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
