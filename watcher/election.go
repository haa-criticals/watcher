package watcher

import (
	"errors"
	"log"
	"time"
)

type ElectionRequest struct {
	Requester *NodeInfo
	Leader    *NodeInfo
	LastBeat  time.Time
	StartedAt time.Time
}

type VoteResponse struct {
	Granted bool
	Term    int64
}

type election struct {
	nodes     []*NodeInfo
	startedAt time.Time
}

func newElection(nodes []*NodeInfo) *election {
	log.Printf("Starting election with %d nodes", len(nodes))
	return &election{
		nodes: nodes,
	}
}

func (e *election) start() error {
	if len(e.nodes) == 0 {
		return errors.New("no nodes to start election")
	}
	e.startedAt = time.Now()
	return nil
}
