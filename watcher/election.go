package watcher

import (
	"errors"
	"time"
)

type electionState int

const (
	inProgress electionState = iota
	elected
	rejected
)

type election struct {
	nodes     []*NodeInfo
	startedAt time.Time
	rejected  int
	granted   int
}

func (e *election) onNonGrantedVote() {
	e.rejected++
}

func (e *election) onGrantedVote() {
	e.granted++
}

func (e *election) isCompleted() bool {
	return e.rejected > len(e.nodes)/2 || e.granted > len(e.nodes)/2
}

func (e *election) currentState() electionState {
	state := inProgress
	if e.granted > len(e.nodes)/2 {
		state = elected
	} else if e.rejected > len(e.nodes)/2 {
		state = rejected
	}
	return state
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
