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

package lfs

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/aws/aws-sdk-go/aws"             //nolint:staticcheck
	"github.com/aws/aws-sdk-go/aws/credentials" //nolint:staticcheck
	"github.com/aws/aws-sdk-go/aws/session"     //nolint:staticcheck
	"github.com/aws/aws-sdk-go/service/s3"      //nolint:staticcheck
)

type S3 struct {
	s3                *s3.S3
	signS3            *s3.S3
	basePath          string
	bucket            string
	expire            time.Duration
	checksumAlgorithm string
}

func NewS3(basePath, endpoint, accessKey, secretKey, bucket string, forcePathStyle bool, s3SignEndpoint string) *S3 {
	sess := session.Must(session.NewSession(&aws.Config{
		Endpoint:         &endpoint,
		Region:           aws.String("us-east-1"),
		Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
		S3ForcePathStyle: &forcePathStyle,
	}))

	if s3SignEndpoint == "" {
		s3SignEndpoint = endpoint
	}

	signSess := session.Must(session.NewSession(&aws.Config{
		Endpoint:         &s3SignEndpoint,
		Region:           aws.String("us-east-1"),
		Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
		S3ForcePathStyle: &forcePathStyle,
	}))

	return &S3{
		basePath:          basePath,
		s3:                s3.New(sess),
		signS3:            s3.New(signSess),
		bucket:            bucket,
		expire:            15 * time.Minute,
		checksumAlgorithm: "SHA256",
	}
}

func (s *S3) SignGet(oid string) (string, error) {
	key := path.Join(s.basePath, transformKey(oid))
	req, _ := s.signS3.GetObjectRequest(&s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	urlStr, err := req.Presign(s.expire)
	if err != nil {
		return "", err
	}
	return urlStr, nil
}

func hexToBase64(hexStr string) (string, error) {
	bin, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bin), nil
}

func (s *S3) SignPut(oid string) (string, error) {
	sha256, err := hexToBase64(oid)
	if err != nil {
		return "", err
	}
	key := path.Join(s.basePath, transformKey(oid))
	req, _ := s.signS3.PutObjectRequest(&s3.PutObjectInput{
		Bucket:            &s.bucket,
		Key:               &key,
		ChecksumAlgorithm: &s.checksumAlgorithm,
		ChecksumSHA256:    &sha256,
	})
	urlStr, err := req.Presign(s.expire)
	if err != nil {
		return "", err
	}
	return urlStr, nil
}

func (s *S3) Put(oid string, r io.Reader, size int64) error {
	sha256, err := hexToBase64(oid)
	if err != nil {
		return err
	}

	key := path.Join(s.basePath, transformKey(oid))
	req, _ := s.s3.PutObjectRequest(&s3.PutObjectInput{
		Bucket:            &s.bucket,
		Key:               &key,
		ContentLength:     &size,
		ChecksumAlgorithm: &s.checksumAlgorithm,
		ChecksumSHA256:    &sha256,
	})
	urlStr, err := req.Presign(s.expire)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequest(http.MethodPut, urlStr, r)
	if err != nil {
		return err
	}
	httpReq.ContentLength = size
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to upload object, status code: %d", resp.StatusCode)
	}
	return nil
}

func (s *S3) Info(oid string) (os.FileInfo, error) {
	key := path.Join(s.basePath, transformKey(oid))
	output, err := s.s3.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFoundError(err) {
			return nil, os.ErrNotExist
		}
		return nil, err
	}
	return &s3FileInfo{
		key:          key,
		size:         *output.ContentLength,
		lastModified: *output.LastModified,
	}, nil
}

type s3FileInfo struct {
	key          string
	size         int64
	lastModified time.Time
}

func (f *s3FileInfo) Name() string {
	return f.key
}

func (f *s3FileInfo) Size() int64 {
	return f.size
}

func (f *s3FileInfo) Mode() os.FileMode {
	return 0444
}

func (f *s3FileInfo) ModTime() (t time.Time) {
	return f.lastModified
}

func (f *s3FileInfo) IsDir() bool {
	return false
}

func (f *s3FileInfo) Sys() interface{} {
	return nil
}

func isNotFoundError(err error) bool {
	if aerr, ok := err.(s3.RequestFailure); ok {
		if aerr.StatusCode() == 404 {
			return true
		}
	}
	return false
}
