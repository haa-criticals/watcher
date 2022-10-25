package monitor

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com.haa-criticals/watcher/watcher"
)

type ErrorHandler interface {
	OnHeartBeatError(err error, watcher *watcher.NodeInfo)
	OnHealthCheckError(err error)
}

type defaultErrorHandler struct{}

func (d *defaultErrorHandler) OnHeartBeatError(err error, watcher *watcher.NodeInfo) {
	log.Printf("error sending heart Beat to %s: %v", watcher.Address, err)
}

func (d *defaultErrorHandler) OnHealthCheckError(err error) {
	log.Printf("error sending health check: %v", err)
}

type Monitor struct {
	errorHandler      ErrorHandler
	notifier          Notifier
	watchers          []*watcher.NodeInfo
	healthChecker     *healthChecker
	doneHeartBeat     chan struct{}
	isHeartBeating    bool
	heartBeatLock     sync.Mutex
	heartBeatInterval time.Duration
}

func (m *Monitor) RegisterWatcher(watcher *watcher.NodeInfo) {
	m.watchers = append(m.watchers, watcher)
	if !m.isHeartBeating {
		m.StartHeartBeating()
	}
}

func (m *Monitor) StartHeartBeating() {
	m.heartBeatLock.Lock()

	if m.isHeartBeating {
		m.heartBeatLock.Unlock()
		return
	}
	m.isHeartBeating = true
	m.heartBeatLock.Unlock()

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
	if len(m.watchers) == 0 {
		return fmt.Errorf("no watchers registered")
	}

	wg := &sync.WaitGroup{}
	wg.Add(len(m.watchers))
	for _, w := range m.watchers {
		go m.sendBeatToWatcher(wg, w)
	}
	wg.Wait()
	return nil
}

func (m *Monitor) sendBeatToWatcher(wg *sync.WaitGroup, watcher *watcher.NodeInfo) {
	defer wg.Done()
	err := m.notifier.Beat(watcher.Address)
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
