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

package s3fs

import (
	"context"
	"fmt"
	"os"

	"github.com/matrixhub-ai/matrixhub/internal/utils"
)

// Mount mounts the specified S3 bucket to the given mount point using s3fs.
func Mount(ctx context.Context, point, endpoint, accessKey, secretKey, bucket, path string, usePathStyle bool) error {
	if err := os.MkdirAll(point, 0755); err != nil {
		return err
	}

	remote := bucket
	if path != "" {
		remote = fmt.Sprintf("%s:%s", bucket, path)
	}

	args := []string{
		remote,
		point,
	}

	if path != "" {
		args = append(args, "-o", "compat_dir")
	}

	if accessKey != "" {
		fs, err := os.OpenFile("./passwd-s3fs", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return err
		}

		_, err = fs.WriteString(accessKey + ":" + secretKey + "\n")
		if err != nil {
			return err
		}
		_ = fs.Close()

		args = append(args, "-o", "passwd_file="+fs.Name())
	}

	if endpoint != "" {
		args = append(args, "-o", "url="+endpoint)
	}
	if usePathStyle {
		args = append(args, "-o", "use_path_request_style")
	}

	cmd := utils.Command(ctx, "s3fs", args...)
	if output, err := cmd.Output(); err != nil {
		return fmt.Errorf("s3fs mount error: %v, output: %s", err, string(output))
	}

	return nil
}

// Unmount unmounts the S3 bucket from the given mount point.
func Unmount(ctx context.Context, point string) error {
	cmd := utils.Command(ctx, "umount", point)
	if output, err := cmd.Output(); err != nil {
		return fmt.Errorf("s3fs unmount error: %v, output: %s", err, string(output))
	}
	return nil
}
