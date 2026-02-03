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

package server

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
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/matrixhub-ai/matrixhub/pkg/config"
	"github.com/matrixhub-ai/matrixhub/pkg/db"
	"github.com/matrixhub-ai/matrixhub/pkg/log"
	"github.com/matrixhub-ai/matrixhub/pkg/server/handler"
	"github.com/matrixhub-ai/matrixhub/pkg/server/handler/project"
	"github.com/matrixhub-ai/matrixhub/pkg/server/middleware"
)

const maxGrpcMsgSize = 100 * 1024 * 1024

type Server struct {
	debug      bool
	cmux       cmux.CMux
	httpServer *http.Server
	engine     *gin.Engine
	gatewayMux *runtime.ServeMux
	grpcServer *grpc.Server
	port       int

	datebase *db.RawDB

	handlers []handler.Registry
}

func NewServer(config *config.Config) (*Server, error) {
	if config == nil {
		return nil, errors.New("config is nil")
	}

	datebase, err := db.New(config.Database.Driver, config.Database.DSN, config.Database.Debug)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}

	server := &Server{
		debug: config.Debug,
		port:  config.APIServer.Port,
		handlers: []handler.Registry{
			project.NewProjectHandler(datebase),
		},
	}

	server.datebase = datebase

	engine := gin.New()
	engine.Use(
		gin.Recovery(),
	)

	server.engine = engine
	// Create the main listener.
	addr := fmt.Sprintf(":%d", server.port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %v", addr, err)
	}

	server.cmux = cmux.New(l)
	server.gatewayMux = runtime.NewServeMux(
		runtime.WithForwardResponseOption(middleware.ResponseHeaderLocation),
		runtime.WithOutgoingHeaderMatcher(func(s string) (string, bool) {
			if s == "Content-Disposition" {
				return s, true
			}
			return fmt.Sprintf("%s%s", runtime.MetadataHeaderPrefix, s), true
		}),
	)

	streamMiddleware := []grpc.StreamServerInterceptor{
		grpc_recovery.StreamServerInterceptor(),
	}
	unaryMiddleware := []grpc.UnaryServerInterceptor{
		grpc_recovery.UnaryServerInterceptor(),
	}

	server.grpcServer = grpc.NewServer(
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			streamMiddleware...,
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			unaryMiddleware...,
		)),
		// gRPC default maximum send message is 4M, change to maximum send message 100M
		grpc.MaxSendMsgSize(maxGrpcMsgSize),
		grpc.MaxRecvMsgSize(maxGrpcMsgSize),
	)
	server.httpServer = &http.Server{
		Handler:           server.engine,
		ReadHeaderTimeout: 30 * time.Second,
	}

	err = server.registerRoutersAndGRPCHandlers()
	if err != nil {
		return nil, fmt.Errorf("failed to register routers and grpc handlers: %v", err)
	}
	return server, nil
}

func (server *Server) registerRoutersAndGRPCHandlers() error {
	// healthz endpoint
	server.engine.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "OK") })

	// register routers
	server.engine.Any("/api/v1alpha1/*any", gin.WrapF(server.gatewayMux.ServeHTTP))

	options := &handler.ServerOptions{
		Router:     server.engine,
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
		err := h.Register(options)
		if err != nil {
			return err
		}
	}

	return nil
}

func (server *Server) Run() <-chan error {
	errorCh := make(chan error, 1)

	grpcL := server.cmux.Match(cmux.HTTP2())
	httpL := server.cmux.Match(cmux.HTTP1Fast())

	go func() {
		log.Infow("Internal http server is listening", "addr", httpL.Addr())
		if err := server.httpServer.Serve(httpL); err != nil {
			errorCh <- err
			if errors.Is(err, http.ErrServerClosed) {
				log.Infow("http server closed")
				return
			}
			log.Errorw("run http server failed", "error", err)
		}
	}()

	go func() {
		log.Infow("Internal grpc server is listening", "addr", grpcL.Addr())
		if err := server.grpcServer.Serve(grpcL); err != nil {
			errorCh <- err
			if errors.Is(err, http.ErrServerClosed) {
				log.Infow("grpc server closed")
				return
			}
			log.Errorw("run grpc server failed", "error", err)
		}
	}()

	go func() {
		log.Infow("api server is listening", "port", server.port)
		if err := server.cmux.Serve(); err != nil {
			errorCh <- err
			if errors.Is(err, http.ErrServerClosed) {
				log.Infow("api server closed")
				return
			}
			log.Errorw("run api server failed", "error", err)
		}
	}()

	return errorCh
}

func (server *Server) Shutdown() {
	log.Infow("api server shutdown")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	server.cmux.Close()

	if err := server.httpServer.Shutdown(ctx); err != nil {
		log.Errorw("shutdown error", "error", err)
	}

	server.grpcServer.GracefulStop()
}
