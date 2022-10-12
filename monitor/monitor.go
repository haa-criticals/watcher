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
	errorHandler    ErrorHandler
	notifier        Notifier
	watchers        []*Watcher
	doneHealthCheck chan struct{}
	doneHeartBeat   chan struct{}
	heartBeatLock   sync.Mutex
	healthCheckLock sync.Mutex
}

func (m *Monitor) RegisterWatcher(watcher *Watcher) {
	m.watchers = append(m.watchers, watcher)
}

func (m *Monitor) StartHeartBeating(period time.Duration) {
	if !m.heartBeatLock.TryLock() {
		return
	}
	if m.doneHeartBeat != nil {
		return
	}
	m.doneHeartBeat = make(chan struct{})
	m.heartBeatLock.Unlock()

	t := time.NewTicker(period)
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
	if !m.healthCheckLock.TryLock() {
		return
	}

	if m.doneHealthCheck != nil {
		return
	}

	m.doneHealthCheck = make(chan struct{})
	m.healthCheckLock.Unlock()

	t := time.NewTicker(second)
	for {
		select {
		case <-t.C:
			err := m.healthCheck(endpoint)
			if err != nil {
				log.Printf("error sending health check: %v", err)
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
	if m.doneHeartBeat != nil {
		close(m.doneHeartBeat)
	}
	if m.doneHealthCheck != nil {
		close(m.doneHealthCheck)
	}
}

func New(options ...Option) *Monitor {
	m := &Monitor{
		doneHealthCheck: make(chan struct{}),
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
