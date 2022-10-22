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
	OnLeaderDown                      func(info *NodeInfo, lastReceivedBeat time.Time)
	nodes                             []*NodeInfo
	registrationKey                   string
	client                            Client
	Address                           string
	registerLocker                    sync.Locker
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
			if time.Now().Sub(w.lastReceivedBeat) > w.maxLeaderAliveInterval {
				w.onNoReceivedHeartBeat()
			}
		case <-w.doneHeartBeatChecking:
			t.Stop()
			return
		}
	}
}

func (w *Watcher) StopHeartBeatChecking() {
	if w.checkingHeartBeat {
		w.doneHeartBeatChecking <- struct{}{}
		w.checkingHeartBeat = false
	}
}

func (w *Watcher) OnReceiveHeartBeat(heartBeatTime time.Time) {
	w.lastReceivedBeat = heartBeatTime
}

func (w *Watcher) RegisterLeader(leader *NodeInfo) {
	w.leader = leader
}

func (w *Watcher) onNoReceivedHeartBeat() {
	if time.Now().Sub(w.lastLeaderDownNotificationTime) > w.minLeaderDownNotificationInterval {
		w.lastLeaderDownNotificationTime = time.Now()
		w.OnLeaderDown(w.leader, w.lastReceivedBeat)
	}
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
