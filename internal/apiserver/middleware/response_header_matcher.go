package middleware

import (
	"context"
	"net/http"

	"google.golang.org/protobuf/proto"
)

// grpc-gateway 中进行 302 重定向
// https://stackoverflow.com/questions/49878855/how-to-do-a-302-redirect-in-grpc-gateway
func ResponseHeaderLocation(ctx context.Context, w http.ResponseWriter, resp proto.Message) error {
	headers := w.Header()
	if location, ok := headers["Grpc-Metadata-Location"]; ok {
		w.Header().Set("Location", location[0])
		w.WriteHeader(http.StatusFound)
	}
	return nil
}
