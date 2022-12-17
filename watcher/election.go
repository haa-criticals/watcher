package watcher

import (
	"errors"
	"time"
)

type election struct {
	nodes     []*NodeInfo
	startedAt time.Time
}

func newElection(nodes []*NodeInfo) (*election, error) {
	if len(nodes) == 0 {
		return nil, errors.New("no nodes to start election")
	}
	return &election{
		nodes:     nodes,
		startedAt: time.Now(),
	}, nil
}
