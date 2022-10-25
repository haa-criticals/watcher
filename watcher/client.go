package watcher

import (
	"context"
	"time"
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
	RequestElection(background context.Context, node, to, leader *NodeInfo, beat time.Time) (*ElectionResponse, error)
	ElectionStart(ctx context.Context, node, to *NodeInfo, priority int32) error
	RequestElectionRegistration(ctx context.Context, node, to *NodeInfo, priority int32) error
	SendElectionVote(ctx context.Context, node, elected, to *NodeInfo) error
	SendElectionConclusion(ctx context.Context, node, elected, to *NodeInfo) error
}
