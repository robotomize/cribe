package botstate

import "context"

func NewNoopBackend() *NoopBackend {
	return &NoopBackend{}
}

var _ Backend = (*NoopBackend)(nil)

type NoopBackend struct{}

func (n NoopBackend) Get(_ context.Context, _ string) ([]byte, error) {
	return nil, nil
}

func (n NoopBackend) Set(_ context.Context, _ string, _ []byte) error {
	return nil
}

func (n NoopBackend) Delete(_ context.Context, _ string) error {
	return nil
}
