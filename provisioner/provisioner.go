package provisioner

import "context"

type Provider interface {
	Create(ctx context.Context) error
	Destroy(ctx context.Context) error
}

type Manager struct {
	Provider Provider
}

func New() *Manager {
	return &Manager{Provider: &Gitlab{}}
}

func WithProvider(provider Provider) *Manager {
	return &Manager{Provider: provider}
}
