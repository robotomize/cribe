package botstate

import "context"

var _ Backend = (*InMemoryBackend)(nil)

func NewInMemoryBackend() *InMemoryBackend {
	return &InMemoryBackend{}
}

type InMemoryBackend struct{}

func (i InMemoryBackend) Get(ctx context.Context, k string) ([]byte, error) {
	panic("implement me")
}

func (i InMemoryBackend) Set(ctx context.Context, k string, v []byte) error {
	panic("implement me")
}

func (i InMemoryBackend) Delete(ctx context.Context, k string) error {
	panic("implement me")
}
