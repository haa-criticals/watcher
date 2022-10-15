package watcher

import (
	"errors"
	"github.com/google/uuid"
	"log"
	"sync"
	"time"
)

type NodeInfo struct {
	ID            uuid.UUID
	BaseURL       string
	electionState electionState
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
	n.ID = uuid.New()
	w.nodes = append(w.nodes, n)
	return w.nodes, nil
}
