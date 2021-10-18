/*
 * Copyright 2021 Seth Vargo
 * Copyright 2021 Mike Helmick
 * Copyright 2021 Maxim Gulimonov
 * Copyright 2020 Ivanov Nikita
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var _ Blob = (*S3)(nil)

func NewS3(cfg S3Config) (*S3, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: &cfg.Region,
		Credentials: credentials.NewStaticCredentials(
			cfg.AccessID, cfg.Secret, "",
		),
	})
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	svc := s3.New(sess)

	return &S3{svc: svc, cfg: cfg}, nil
}

type S3 struct {
	svc *s3.S3
	cfg S3Config
}

func (s *S3) CreateObject(ctx context.Context, bucket, key string, contents []byte) error {
	cacheControl := "public, max-age=86400"

	putInput := s3.PutObjectInput{
		Bucket:       aws.String(bucket),
		Key:          aws.String(key),
		CacheControl: aws.String(cacheControl),
		Body:         bytes.NewReader(contents),
	}

	if _, err := s.svc.PutObjectWithContext(ctx, &putInput); err != nil {
		return fmt.Errorf("create object: %w", err)
	}

	return nil
}

func (s *S3) DeleteObject(ctx context.Context, bucket, key string) error {
	if _, err := s.svc.DeleteObjectWithContext(
		ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		},
	); err != nil {
		return fmt.Errorf("delete object: %w", err)
	}

	return nil
}

func (s *S3) GetObject(ctx context.Context, bucket, key string) ([]byte, error) {
	o, err := s.svc.GetObjectWithContext(
		ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		},
	)
	if err != nil {
		var aerr awserr.Error
		if errors.As(err, &aerr) && (aerr.Code() == s3.ErrCodeNoSuchBucket || aerr.Code() == s3.ErrCodeNoSuchKey) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("get object: %w", err)
	}

	defer o.Body.Close()

	b, err := io.ReadAll(o.Body)
	if err != nil {
		return nil, fmt.Errorf("read object: %w", err)
	}

	return b, nil
}

func (s *S3) PublicAccess(_ context.Context, bucket, key string) string {
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, s.cfg.Region, key)
}
