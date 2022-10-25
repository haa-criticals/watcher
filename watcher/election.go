package watcher

import (
	"errors"
	"time"
)

type electionState int

const (
	requested electionState = iota
	accepted
	registered
	voted
	finished
	failed
)

type ElectionRequest struct {
	Requester *NodeInfo
	Leader    *NodeInfo
	LastBeat  time.Time
	StartedAt time.Time
}

type Election struct {
	nodes            []*NodeInfo
	state            electionState
	OnStartElection  func(nodes []*NodeInfo)
	OnNewLeaderElect func(nodes *NodeInfo)
	newLeader        *NodeInfo
	startedAt        time.Time
}

func NewElection(nodes []*NodeInfo) *Election {
	return &Election{
		nodes:     nodes,
		state:     requested,
		startedAt: time.Now(),
	}
}

func (e *Election) Start() error {
	if len(e.nodes) == 0 {
		return errors.New("no nodes to start election")
	}
	if !e.CheckNodesAccepted() {
		return errors.New("not all nodes accepted election yet")
	}
	if e.OnStartElection != nil {
		e.OnStartElection(e.nodes)
	}
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

func (e *Election) WaitRegistration() {
	t := time.Tick(1 * time.Second)
	for {
		select {
		case <-t:
			if e.checkNodesRegistered() {
				e.newLeader = e.getHighestPriority()
				e.state = registered
				return
			}
		}
	}
}

func (e *Election) ReceivePriority(address string, priority int32) {
	for _, n := range e.nodes {
		if n.Address == address {
			n.electionState = registered
			n.priority = priority
		}
	}
}

func (e *Election) checkNodesRegistered() bool {
	for _, n := range e.nodes {
		if n.electionState != registered {
			return false
		}
	}
	return true
}

func (e *Election) getHighestPriority() *NodeInfo {
	var highest *NodeInfo
	for _, n := range e.nodes {
		if highest == nil || n.priority > highest.priority {
			highest = n
		}
	}
	return highest
}

func (e *Election) WaitVotes() {
	t := time.Tick(1 * time.Second)
	for {
		select {
		case <-t:
			if e.checkNodesVoted() {
				e.newLeader = e.getMostVoted()
				e.OnNewLeaderElect(e.newLeader)
				e.state = voted
				return
			}
		}
	}
}

func (e *Election) checkNodesVoted() bool {
	for _, n := range e.nodes {
		if n.electionState != voted {
			return false
		}
	}
	return true
}

func (e *Election) ReceiveVote(address string, vote string) {
	for _, n := range e.nodes {
		if n.Address == address {
			n.electionVote = vote
			n.electionState = voted
		}
	}
}

func (e *Election) getMostVoted() *NodeInfo {
	votes := make(map[string]int)
	for _, n := range e.nodes {
		votes[n.electionVote]++
	}
	var mostVoted *NodeInfo
	for _, n := range e.nodes {
		if mostVoted == nil || votes[n.Address] > votes[mostVoted.electionVote] {
			mostVoted = n
		}
	}
	return mostVoted
}

func (e *Election) Accepted(n *NodeInfo) {
	n.electionState = accepted
}
