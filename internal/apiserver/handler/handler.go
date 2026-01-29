package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

type ServerOptions struct {
	Router      *gin.Engine
	GatewayMux  *runtime.ServeMux
	GRPCServer  *grpc.Server
	GRPCAddr    string
	GRPCDialOpt []grpc.DialOption
}

type IHandler interface {
	RegisterToServer(*ServerOptions)
}
