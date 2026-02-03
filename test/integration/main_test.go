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

package integration

import (
	"os"
	"strconv"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/pkg/config"
	"github.com/matrixhub-ai/matrixhub/pkg/server"
)

var projectClient v1alpha1.ProjectsClient

func TestMain(m *testing.M) {
	// Setup code before running tests can be added here

	conf, err := config.Init("")
	if err != nil {
		panic(err)
	}
	conf.Database.DSN = "file::memory:?cache=shared"
	server, err := server.NewServer(conf)
	if err != nil {
		panic(err)
	}

	go func() {
		errCh := server.Run()
		err := <-errCh
		if err != nil {
			panic(err)
		}
	}()

	grpcClient, err := grpc.NewClient("127.0.0.1:"+strconv.Itoa(conf.APIServer.Port),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = grpcClient.Close()
	}()

	projectClient = v1alpha1.NewProjectsClient(grpcClient)

	exitCode := m.Run()

	os.Exit(exitCode)
}
