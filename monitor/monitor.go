package monitor

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type ErrorHandler interface {
	OnHeartBeatError(err error, watcher *Watcher)
	OnHealthCheckError(err error)
}

type defaultErrorHandler struct{}

func (d *defaultErrorHandler) OnHeartBeatError(err error, watcher *Watcher) {
	log.Printf("error sending heart Beat to %s: %v", watcher.BaseURL, err)
}

func (d *defaultErrorHandler) OnHealthCheckError(err error) {
	log.Printf("error sending health check: %v", err)
}

type Watcher struct {
	BaseURL string
}

type Monitor struct {
	errorHandler      ErrorHandler
	notifier          Notifier
	watchers          []*Watcher
	doneHealthCheck   chan struct{}
	healthCheckLock   sync.Mutex
	healthCheckErrors int
	isHealthChecking  bool
	doneHeartBeat     chan struct{}
	isHeartBeating    bool
	heartBeatLock     sync.Mutex
	heartBeatInterval time.Duration
}

func (m *Monitor) RegisterWatcher(watcher *Watcher) {
	m.watchers = append(m.watchers, watcher)
}

func (m *Monitor) StartHeartBeating() {
	m.heartBeatLock.Lock()

	if m.isHeartBeating {
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
	for _, watcher := range m.watchers {
		go m.sendBeatToWatcher(wg, watcher)
	}
	wg.Wait()
	return nil
}

func (m *Monitor) sendBeatToWatcher(wg *sync.WaitGroup, watcher *Watcher) {
	defer wg.Done()
	err := m.notifier.Beat(watcher.BaseURL)
	if err != nil {
		log.Printf("error sending heart Beat to %s: %v", watcher.BaseURL, err)
		m.errorHandler.OnHeartBeatError(err, watcher)
	}
}

func (m *Monitor) StartHealthChecks(second time.Duration, endpoint string) {
	m.healthCheckLock.Lock()
	if m.isHealthChecking {
		return
	}
	m.isHealthChecking = true
	m.healthCheckLock.Unlock()

	t := time.NewTicker(second)
	for {
		select {
		case <-t.C:
			err := m.healthCheck(endpoint)
			if err != nil {
				m.healthCheckErrors++
				m.errorHandler.OnHealthCheckError(err)
			} else {
				m.healthCheckErrors = 0
			}
		case <-m.doneHealthCheck:
			t.Stop()
			return
		}
	}
}

func (m *Monitor) healthCheck(endpoint string) error {
	r, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(r)

	if err != nil {
		return err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("error closing response body: %v", err)
		}
	}(res.Body)

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}
	return nil
}

func (m *Monitor) Stop() {
	if m.isHeartBeating {
		m.doneHeartBeat <- struct{}{}
		m.isHeartBeating = false
	}
	if m.isHealthChecking {
		m.doneHealthCheck <- struct{}{}
		m.isHealthChecking = false
	}
}

func (m *Monitor) IsHealthy() bool {
	return m.healthCheckErrors < 3
}

func New(options ...Option) *Monitor {
	m := &Monitor{
		heartBeatInterval: 5 * time.Second,
		doneHeartBeat:     make(chan struct{}),
		doneHealthCheck:   make(chan struct{}),
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
	return m
}
