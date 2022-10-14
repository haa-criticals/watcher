package watcher

import (
	"sync"
	"time"
)

type NodeInfo struct {
	BaseURL string
}

type Watcher struct {
	leader                 NodeInfo
	lastReceivedBeat       time.Time
	checkHeartBeatInterval time.Duration
	maxLeaderAliveInterval time.Duration
	doneHeartBeatChecking  chan struct{}
	OnLeaderDown           func(info NodeInfo, lastReceivedBeat time.Time)
	checkingHeartBeat      bool
	checkingHeartBeatLock  sync.Mutex
}

func (w *Watcher) StartHeartBeatChecking() {
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
				w.OnLeaderDown(w.leader, w.lastReceivedBeat)
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
