package monitor

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com.haa-criticals/watcher/watcher"
)

type ErrorHandler interface {
	OnHeartBeatError(err error, watcher *watcher.Peer)
	OnHealthCheckError(err error)
}

type defaultErrorHandler struct{}

func (d *defaultErrorHandler) OnHeartBeatError(err error, watcher *watcher.Peer) {
	log.Printf("error sending heart Beat to %s: %v", watcher.Address, err)
}

func (d *defaultErrorHandler) OnHealthCheckError(err error) {
	log.Printf("error sending health check: %v", err)
}

type Monitor struct {
	errorHandler      ErrorHandler
	notifier          Notifier
	peers             []*watcher.Peer
	healthChecker     *healthChecker
	doneHeartBeat     chan struct{}
	isHeartBeating    bool
	heartBeatLock     sync.Mutex
	heartBeatInterval time.Duration
	term              int64
	Address           string
}

func (m *Monitor) RegisterWatcher(peer *watcher.Peer) {
	m.peers = append(m.peers, peer)
}

func (m *Monitor) StartHeartBeating() {
	m.heartBeatLock.Lock()

	if m.isHeartBeating {
		m.heartBeatLock.Unlock()
		return
	}
	m.isHeartBeating = true
	m.heartBeatLock.Unlock()

	_ = m.heartBeat()
	t := time.NewTicker(m.heartBeatInterval)
	for {
		select {
		case <-t.C:

			err := m.heartBeat()
			if err != nil {
				log.Printf("error sending heart Beat: %v", err)
			}
		case <-m.doneHeartBeat:
			t.Stop()
			return
		}
	}
}

func (m *Monitor) heartBeat() error {
	if len(m.peers) == 0 {
		return fmt.Errorf("no peers registered")
	}

	wg := &sync.WaitGroup{}
	wg.Add(len(m.peers))
	for _, w := range m.peers {
		go m.sendBeatToWatcher(wg, w)
	}
	wg.Wait()
	return nil
}

func (m *Monitor) sendBeatToWatcher(wg *sync.WaitGroup, watcher *watcher.Peer) {
	defer wg.Done()
	err := m.notifier.Beat(watcher.Address, m.Address, m.term)
	if err != nil {
		log.Printf("error sending heart Beat to %s: %v", watcher.Address, err)
		m.errorHandler.OnHeartBeatError(err, watcher)
	}
}

func (m *Monitor) StartHealthChecks() {
	m.healthChecker.Start()
}

func (m *Monitor) IsHealthy() bool {
	return m.healthChecker.fails < m.healthChecker.maxFails
}

func (m *Monitor) Stop() {
	if m.isHeartBeating {
		m.doneHeartBeat <- struct{}{}
		m.isHeartBeating = false
	}
	m.healthChecker.Stop()
}

func (m *Monitor) IsMonitoring() bool {
	return m.isHeartBeating || m.healthChecker.active
}

func (m *Monitor) NewTerm(term int64) {
	m.term = term
}

func New(options ...Option) *Monitor {
	m := &Monitor{
		heartBeatInterval: 5 * time.Second,
		doneHeartBeat:     make(chan struct{}),
		healthChecker: &healthChecker{
			interval: 5 * time.Second,
			maxFails: 3,
			done:     make(chan struct{}),
		},
	}

	for _, opt := range options {
		opt(m)
	}

	if m.errorHandler == nil {
		m.errorHandler = &defaultErrorHandler{}
	}

	if m.notifier == nil {
		m.notifier = &defaultNotifier{}
	}

	m.healthChecker.errorHandler = m.errorHandler.OnHealthCheckError
	return m
}
