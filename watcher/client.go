package watcher

import (
	"context"
)

type RegisterResponse struct {
	Success bool
	Nodes   []*NodeInfo
}

type ElectionResponse struct {
	Accepted bool
	Node     *NodeInfo
}

type Client interface {
	RequestRegister(ctx context.Context, address, key string) (*RegisterResponse, error)
	AckNode(ctx context.Context, address, key string, node *NodeInfo) (*NodeInfo, error)
}
