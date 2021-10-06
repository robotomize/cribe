package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
)

var ErrNotFound = fmt.Errorf("storage object not found")

var _ Blob = (*S3)(nil)

type S3Config struct {
	Region   string `env:"S3_REGION,default=eu-west-3"`
	AccessID string `env:"S3_ACCESS_ID"`
	Secret   string `env:"S3_SECRET_KEY"`
}

func NewS3(cfg S3Config) (*S3, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      &cfg.Region,
		Credentials: credentials.NewStaticCredentials(cfg.AccessID, cfg.Secret, ""),
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
