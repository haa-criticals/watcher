package provisioner

import "context"

type Provider interface {
	Create(ctx context.Context) error
	Destroy(ctx context.Context) error
}

type Manager struct {
	Provider Provider
}

func (m *Manager) Create(ctx context.Context) error {
	return m.Provider.Create(ctx)
}

func (m *Manager) Destroy(ctx context.Context) error {
	return m.Provider.Destroy(ctx)
}

func New(config *Config) *Manager {
	return &Manager{Provider: &Gitlab{
		config: config,
	}}
}

func WithProvider(provider Provider) *Manager {
	return &Manager{Provider: provider}
}
