package watcher

import (
	"errors"
)

type electionState int

const (
	requested electionState = iota
	accepted
	voting
	finished
)

type Election struct {
	nodes           []*NodeInfo
	state           electionState
	OnStartElection func(nodes []*NodeInfo)
}

func New(nodes []*NodeInfo) *Election {
	return &Election{
		nodes: nodes,
	}
}

func (e *Election) Start() error {
	if len(e.nodes) == 0 {
		return errors.New("no nodes to start election")
	}
	if !e.CheckNodesAccepted() {
		return errors.New("not all nodes accepted election yet")
	}
	e.state = voting
	e.OnStartElection(e.nodes)
	return nil
}

func (e *Election) CheckNodesAccepted() bool {
	for _, n := range e.nodes {
		if n.electionState != accepted {
			return false
		}
	}
	return true
}
