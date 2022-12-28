package watcher

import (
	"context"
	"log"
	"math/rand"
	"sync"
	"time"
)

type Peer struct {
	Address string
}

type Vote struct {
	Adress  string
	Granted bool
	Term    int64
}

type Candidate struct {
	Term    int64
	Address string
}

type Config struct {
	Address                string
	HeartBeatCheckInterval time.Duration
	MaxDelayForElection    int64
}

const (
	leaaderAliveMultiplier         = 3
	leaderDonwNotificationInterval = 9
)

type Watcher struct {
	lastReceivedBeat               time.Time
	doneHeartBeatChecking          chan struct{}
	checkingHeartBeat              bool
	checkingHeartBeatLock          sync.Mutex
	lastLeaderDownNotificationTime time.Time
	OnLeaderDown                   func(nodes []*Peer, lastBeat time.Time)
	nodes                          []*Peer
	client                         Client
	Address                        string
	election                       *election
	maxMillisDelayForElection      int64
	term                           int64
	votedFor                       string
	config                         Config
	OnElectionWon                  func(*Watcher, int64)
	electionLock                   sync.Mutex
	isLeader                       bool
	OnLostLeadership               func(watcher *Watcher, int642 int64)
	electionTimer                  *time.Timer
}

func New(client Client, config Config) *Watcher {
	if config.HeartBeatCheckInterval == 0 {
		config.HeartBeatCheckInterval = 1 * time.Second
	}

	if config.MaxDelayForElection == 0 {
		config.MaxDelayForElection = 1000
	}
	return &Watcher{
		client:                client,
		config:                config,
		doneHeartBeatChecking: make(chan struct{}),
	}
}

func (w *Watcher) RegisterNodes(nodes ...string) {
	for _, node := range nodes {
		w.nodes = append(w.nodes, &Peer{Address: node})
	}
}

func (w *Watcher) StartHeartBeatChecking() {
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
	return time.Now().Sub(w.lastReceivedBeat) < w.config.HeartBeatCheckInterval*leaaderAliveMultiplier
}

func (w *Watcher) StopHeartBeatChecking() {
	if w.checkingHeartBeat {
		w.doneHeartBeatChecking <- struct{}{}
		w.checkingHeartBeat = false
	}
}

func (w *Watcher) OnReceiveHeartBeat(address string, term int64, heartBeatTime time.Time) {
	log.Printf("%s received heartbeat from %s on term %d", w.Address, address, term)
	w.lastReceivedBeat = heartBeatTime
	if w.isLeader && term > w.term {
		w.isLeader = false
		w.term = term
		if w.OnLostLeadership != nil {
			go w.OnLostLeadership(w, term)
		}
	}
	if !w.checkingHeartBeat {
		go w.StartHeartBeatChecking()
	}
}

func (w *Watcher) onNoReceivedHeartBeat() {
	if time.Now().Sub(w.lastLeaderDownNotificationTime) > w.config.HeartBeatCheckInterval*leaderDonwNotificationInterval {
		w.lastLeaderDownNotificationTime = time.Now()
		if w.OnLeaderDown != nil {
			w.OnLeaderDown(w.nodes, w.lastReceivedBeat)
		}
		go w.startElection()
	}
}
func (w *Watcher) startElection() {
	var err error
	w.election, err = newElection(w.nodes, w.term)
	if err != nil {
		log.Println(err)
		return
	}
	t := rand.Int63n(w.config.MaxDelayForElection)
	w.electionTimer = time.AfterFunc(time.Duration(t)*time.Millisecond, w.requestVotes)
}

func (w *Watcher) requestVotes() {
	w.electionLock.Lock()
	defer w.electionLock.Unlock()
	w.electionTimer = nil
	if w.isLeaderAlive() {
		log.Printf("%s Leader is alive, no need to start election", w.Address)
		return
	}
	log.Printf("%s is requesting votes on term %d", w.Address, w.term+1)
	if w.term != w.election.term {
		return
	}
	w.term++
	w.votedFor = w.Address
	w.election.onGrantedVote()
	for _, node := range w.election.nodes {
		go w.requestVote(node)
	}
}

func (w *Watcher) requestVote(node *Peer) {
	vote, err := w.client.RequestVote(context.Background(), node.Address, w.Address, w.term)
	if err != nil || !vote.Granted {
		w.election.onNonGrantedVote()
	} else {
		w.election.onGrantedVote()
	}

	switch w.election.currentState() {
	case elected:
		w.election.finished = true
		w.onElected()
	case rejected:
		w.election.finished = true
	}
}

func (w *Watcher) onElected() {
	w.StopHeartBeatChecking()
	w.isLeader = true
	if w.OnElectionWon != nil {
		w.OnElectionWon(w, w.term)
	}
}

func (w *Watcher) LastReceivedBeat() time.Time {
	return w.lastReceivedBeat
}

func (w *Watcher) OnReceiveVoteRequest(request *Candidate) *Vote {
	w.electionLock.Lock()
	defer w.electionLock.Unlock()
	if w.electionTimer != nil {
		w.electionTimer.Stop()
		w.electionTimer = nil
	}
	if request.Term < w.term || (w.term == request.Term && w.votedFor != "") {
		log.Printf("%s on term %d Rejected vote request from %s on term %d", w.Address, w.term, request.Address, request.Term)
		return &Vote{
			Granted: false,
			Term:    w.term,
		}
	}
	log.Printf("%s on term %d granted vote to %s on term %d", w.Address, w.term, request.Address, request.Term)
	w.votedFor = request.Address
	w.term = request.Term
	return &Vote{
		Granted: true,
		Term:    w.term,
	}
}

func (w *Watcher) IsLeader() bool {
	return w.isLeader
}
