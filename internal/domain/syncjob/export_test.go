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

package syncjob

// ModelCacheOptions are the options of model cache jobs.
type ModelCacheOptions = modelCacheOptions

// SetModelCacheOptions tunes the model cache jobs that svc runs, so that tests
// need not wait as long as real downloads do.
func SetModelCacheOptions(svc ISyncJobService, opts ModelCacheOptions) {
	svc.(*SyncJobService).modelCache = opts.withDefaults()
}
