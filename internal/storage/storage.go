package storage

import "context"

type Blob interface {
	CreateObject(ctx context.Context, bucket, key string, contents []byte) error
	DeleteObject(ctx context.Context, bucket, key string) error
	GetObject(ctx context.Context, bucket, key string) ([]byte, error)
}
