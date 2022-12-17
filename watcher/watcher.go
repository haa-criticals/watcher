package watcher

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"sync"
	"time"
)

type NodeInfo struct {
	Address string
}

type Vote struct {
	Adress  string
	Granted bool
	Term    int64
}

type Candidate struct {
	Term     int64
	Address  string
	Priority int32
}

type Config struct {
	RegistrationKey                string
	Address                        string
	Priority                       int32
	HeartBeatCheckInterval         time.Duration
	LeaderAliveInterval            time.Duration
	LeaderDownNotificationInterval time.Duration
	MaxDelayForElection            int64
}

type Watcher struct {
	leader                         *NodeInfo
	lastReceivedBeat               time.Time
	doneHeartBeatChecking          chan struct{}
	checkingHeartBeat              bool
	checkingHeartBeatLock          sync.Mutex
	lastLeaderDownNotificationTime time.Time
	OnLeaderDown                   func(leader *NodeInfo, nodes []*NodeInfo, lastBeat time.Time)
	nodes                          []*NodeInfo
	client                         Client
	Address                        string
	registerLocker                 sync.Locker
	election                       *election
	maxMillisDelayForElection      int64
	term                           int64
	votedFor                       string
	config                         Config
}

func New(client Client, config Config) *Watcher {
	if config.HeartBeatCheckInterval == 0 {
		config.HeartBeatCheckInterval = 1 * time.Second
	}

	if config.LeaderAliveInterval == 0 {
		config.LeaderAliveInterval = 15 * time.Second
	}

	if config.LeaderDownNotificationInterval == 0 {
		config.LeaderDownNotificationInterval = 20 * time.Second
	}

	if config.MaxDelayForElection == 0 {
		config.MaxDelayForElection = 1000
	}
	return &Watcher{
		client:                client,
		config:                config,
		doneHeartBeatChecking: make(chan struct{}),
		registerLocker:        &sync.Mutex{},
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

	t := time.NewTicker(w.config.HeartBeatCheckInterval)
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
	return time.Now().Sub(w.lastReceivedBeat) < w.config.LeaderAliveInterval
}

func (w *Watcher) RegisterNode(n *NodeInfo, key string) ([]*NodeInfo, error) {
	if key != w.config.RegistrationKey {
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
	if key != w.config.RegistrationKey {
		return errors.New("invalid registration key")
	}
	w.registerLocker.Lock()
	w.nodes = append(w.nodes, info)
	w.registerLocker.Unlock()
	log.Printf("Acked node %s", info.Address)
	log.Printf("Registered nodes: %v", w.nodes)
	return nil
}

func (w *Watcher) StopHeartBeatChecking() {
	if w.checkingHeartBeat {
		w.doneHeartBeatChecking <- struct{}{}
		w.checkingHeartBeat = false
	}
}

func (w *Watcher) OnReceiveHeartBeat(heartBeatTime time.Time) {
	w.lastReceivedBeat = heartBeatTime
	if !w.checkingHeartBeat {
		go w.StartHeartBeatChecking()
	}
	log.Printf("Received heart beat from %v", w.leader)
}

func (w *Watcher) RegisterLeader(leader *NodeInfo) {
	w.leader = leader
}

func (w *Watcher) onNoReceivedHeartBeat() {
	if time.Now().Sub(w.lastLeaderDownNotificationTime) > w.config.LeaderDownNotificationInterval {
		w.lastLeaderDownNotificationTime = time.Now()
		if w.OnLeaderDown != nil {
			w.OnLeaderDown(w.leader, w.nodes, w.lastReceivedBeat)
		}
		go w.startElection()
	}
}
func (w *Watcher) startElection() {
	t := rand.Int63n(w.config.MaxDelayForElection)
	time.AfterFunc(time.Duration(t)*time.Millisecond, w.requestVotes)
}

func (w *Watcher) requestVotes() {
	var err error
	w.election, err = newElection(w.nodes)
	if err != nil {
		log.Println(err)
		return
	}
	w.term++
	w.votedFor = w.Address
	for _, node := range w.election.nodes {
		go w.requestVote(node)
	}
}

func (w *Watcher) requestVote(node *NodeInfo) {
	w.client.RequestVote(context.Background(), node.Address, w.Address, w.term)
}

func (w *Watcher) LastReceivedBeat() time.Time {
	return w.lastReceivedBeat
}

func (w *Watcher) OnNewLeader(leader *NodeInfo) {
	for i := range w.nodes {
		if w.nodes[i].Address == leader.Address {
			w.nodes[i] = w.leader
			break
		}
	}
	w.leader = leader
	if w.leader.Address == w.Address {
		w.StartHeartBeatChecking()
	}
}

func (w *Watcher) OnReceiveVoteRequest(request *Candidate) *Vote {
	if request.Term < w.term || w.config.Priority > request.Priority || (w.term == request.Term && w.votedFor != "") {
		return &Vote{
			Granted: false,
			Term:    w.term,
		}
	}
	w.votedFor = request.Address
	w.term = request.Term
	return &Vote{
		Granted: true,
		Term:    w.term,
	}
}
