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

package jobserver

import (
	"context"
	"log/slog"
	"time"

	"github.com/matrixhub-ai/matrixhub/internal/domain/artifactscan"
)

type ArtifactScanWorker struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func NewArtifactScanWorker(service *artifactscan.Service) *ArtifactScanWorker {
	ctx, cancel := context.WithCancel(context.Background())
	worker := &ArtifactScanWorker{cancel: cancel, done: make(chan struct{})}
	go func() {
		defer close(worker.done)
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := service.ProcessNext(ctx); err != nil && ctx.Err() == nil {
					slog.Error("artifact scan processor failed", "error", err)
				}
			}
		}
	}()
	return worker
}
func (w *ArtifactScanWorker) Close() { w.cancel(); <-w.done }
