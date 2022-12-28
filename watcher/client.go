package watcher

import (
	"context"
)

type RegisterResponse struct {
	Success bool
	Nodes   []*Peer
}

type Client interface {
	RequestVote(ctx context.Context, address, candidate string, term int64) (*Vote, error)
}
