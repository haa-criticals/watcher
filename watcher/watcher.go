package watcher

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"
)

type NodeInfo struct {
	Address       string
	electionState electionState
	priority      int
	electionVote  string
}

type Watcher struct {
	leader                            *NodeInfo
	lastReceivedBeat                  time.Time
	checkHeartBeatInterval            time.Duration
	maxLeaderAliveInterval            time.Duration
	doneHeartBeatChecking             chan struct{}
	checkingHeartBeat                 bool
	checkingHeartBeatLock             sync.Mutex
	lastLeaderDownNotificationTime    time.Time
	minLeaderDownNotificationInterval time.Duration
	OnLeaderDown                      func(leader *NodeInfo, nodes []*NodeInfo, lastBeat time.Time)
	nodes                             []*NodeInfo
	registrationKey                   string
	client                            Client
	Address                           string
	registerLocker                    sync.Locker
	election                          *Election
	priority                          int32
}

func New(client Client) *Watcher {
	return &Watcher{
		client:                            client,
		checkHeartBeatInterval:            1 * time.Second,
		maxLeaderAliveInterval:            5 * time.Second,
		minLeaderDownNotificationInterval: 5 * time.Second,
		doneHeartBeatChecking:             make(chan struct{}),
		registerLocker:                    &sync.Mutex{},
	}
}

func (w *Watcher) StartHeartBeatChecking() {
	if w.leader == nil {
		log.Println("There is no elected leader, can't start heart beat checking")
		return
	}

	w.checkingHeartBeatLock.Lock()
	if w.checkingHeartBeat {
		w.checkingHeartBeatLock.Unlock()
		return
	}

	w.checkingHeartBeat = true
	w.checkingHeartBeatLock.Unlock()

	t := time.NewTicker(w.checkHeartBeatInterval)
	for {
		select {
		case <-t.C:
			if !w.isLeaderAlive() {
				w.onNoReceivedHeartBeat()
			}
		case <-w.doneHeartBeatChecking:
			t.Stop()
			return
		}
	}
}

func (w *Watcher) isLeaderAlive() bool {
	return time.Now().Sub(w.lastReceivedBeat) < w.maxLeaderAliveInterval
}

func (w *Watcher) StopHeartBeatChecking() {
	if w.checkingHeartBeat {
		w.doneHeartBeatChecking <- struct{}{}
		w.checkingHeartBeat = false
	}
}

func (w *Watcher) OnReceiveHeartBeat(heartBeatTime time.Time) {
	w.lastReceivedBeat = heartBeatTime
	log.Printf("Received heart beat from %v", w.leader)
}

func (w *Watcher) RegisterLeader(leader *NodeInfo) {
	w.leader = leader
}

func (w *Watcher) onNoReceivedHeartBeat() {
	if time.Now().Sub(w.lastLeaderDownNotificationTime) > w.minLeaderDownNotificationInterval {
		w.lastLeaderDownNotificationTime = time.Now()
		if w.OnLeaderDown != nil {
			w.OnLeaderDown(w.leader, w.nodes, w.lastReceivedBeat)
		}
		w.requestElection()
	}
}

func (w *Watcher) requestElection() {
	w.election = NewElection(w.nodes)
	node := &NodeInfo{
		Address: w.Address,
	}
	for _, n := range w.nodes {
		res, err := w.requestElectionStart(node, n, w.lastReceivedBeat)
		if err != nil || !res.Accepted {
			w.election.state = failed
			return
		}
		w.election.Accepted(n)
	}

	for _, n := range w.nodes {
		err := w.client.ElectionStart(context.Background(), node, n, w.priority)
		if err != nil {
			w.election.state = failed
			return
		}
	}

	w.election.WaitRegistration()
	w.election.WaitVotes()

	for _, n := range w.nodes {
		err := w.client.SendElectionConclusion(context.Background(), node, n, w.election.newLeader)
		if err != nil {
			w.election.state = failed
			return
		}
	}
}

func (w *Watcher) requestElectionStart(node, to *NodeInfo, beat time.Time) (*ElectionResponse, error) {
	log.Printf("Requesting election from %s", to.Address)
	res, err := w.client.RequestElection(context.Background(), node, to, w.leader, beat)
	if err != nil {
		log.Printf("Failed to request election to %s: %s", to.Address, err)
		return nil, err
	}
	return res, nil
}

func (w *Watcher) OnNewElectionRequest(_ context.Context, request *ElectionRequest) (*ElectionResponse, error) {
	node := &NodeInfo{
		Address: w.Address,
	}
	if w.leader == nil {
		return &ElectionResponse{
			Accepted: false,
			Node:     node,
		}, errors.New("there is no leader")
	}
	if w.leader.Address != request.Leader.Address {
		log.Printf("Leader is not %s, can't handle election request", request.Leader.Address)
		return &ElectionResponse{
			Accepted: false,
			Node:     node,
		}, errors.New("leader is not " + request.Leader.Address)
	}

	if request.LastBeat.Before(w.lastReceivedBeat) && w.isLeaderAlive() {
		log.Printf("Leader %s is alive, can't handle election request", request.Leader.Address)
		return &ElectionResponse{
			Accepted: false,
			Node:     node,
		}, errors.New("leader is alive")
	}

	if w.election != nil && w.election.startedAt.Before(request.StartedAt) {
		return &ElectionResponse{
			Accepted: false,
			Node:     node,
		}, errors.New("the Election has been already started")
	}

	w.election = NewElection(w.nodes)
	w.election.startedAt = request.StartedAt

	return &ElectionResponse{
		Accepted: true,
		Node:     node,
	}, nil

}

func (w *Watcher) RegisterNode(n *NodeInfo, key string) ([]*NodeInfo, error) {
	if key != w.registrationKey {
		return nil, errors.New("invalid registration key")
	}
	w.registerLocker.Lock()
	defer w.registerLocker.Unlock()
	w.nodes = append(w.nodes, n)
	return w.nodes, nil
}

func (w *Watcher) RequestRegister(endpoint, key string) error {
	log.Printf("Requesting registration to %s", endpoint)
	res, err := w.client.RequestRegister(context.Background(), endpoint, key)
	if err != nil {
		return err
	}
	if !res.Success {
		return errors.New("failed to register node")
	}

	w.leader = &NodeInfo{
		Address: endpoint,
	}

	for _, n := range res.Nodes {
		if n.Address == w.Address {
			continue
		}
		w.nodes = append(w.nodes, &NodeInfo{Address: n.Address})
		go func(n *NodeInfo) {
			log.Printf("Requesting ack from %s", n.Address)
			_, err := w.client.AckNode(context.Background(), n.Address, key, &NodeInfo{Address: w.Address})
			if err != nil {
				log.Printf("Failed to ack node %s: %s", n.Address, err)
			}
		}(n)
	}
	log.Printf("Registered to %s", endpoint)
	return nil
}

func (w *Watcher) AckNode(info *NodeInfo, key string) error {
	log.Printf("Received ack from %s", info.Address)
	if key != w.registrationKey {
		return errors.New("invalid registration key")
	}
	w.registerLocker.Lock()
	w.nodes = append(w.nodes, info)
	w.registerLocker.Unlock()
	log.Printf("Acked node %s", info.Address)
	log.Printf("Registered nodes: %v", w.nodes)
	return nil
}

func (w *Watcher) LastReceivedBeat() time.Time {
	return w.lastReceivedBeat
}
