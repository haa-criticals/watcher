package watcher

import "context"

type RegisterResponse struct {
	Success bool
	Id      string
	Nodes   []*NodeInfo
}

type Client interface {
	RequestRegister(ctx context.Context, address, key string) (*RegisterResponse, error)
}
