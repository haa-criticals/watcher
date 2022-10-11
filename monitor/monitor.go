package monitor

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type Watcher struct {
	BaseURL string
}

type Monitor struct {
	notifier              Notifier
	watchers              []*Watcher
	HeartBeatErrorHandler func(error, *Watcher)
	done                  chan struct{}
}

func (m *Monitor) RegisterWatcher(watcher *Watcher) {
	m.watchers = append(m.watchers, watcher)
}

func (m *Monitor) StartHeartBeating(period time.Duration) {
	t := time.NewTicker(period)
	for {
		select {
		case <-t.C:
			err := m.heartBeat()
			if err != nil {
				log.Printf("error sending heart Beat: %v", err)
			}
		case <-m.done:
			t.Stop()
			return
		}
	}
}

func (m *Monitor) StopHeartBeat() {
	m.done <- struct{}{}
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
		if m.HeartBeatErrorHandler != nil {
			m.HeartBeatErrorHandler(err, watcher)
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

func New(notifier Notifier) *Monitor {
	return &Monitor{
		notifier: notifier,
		done:     make(chan struct{}),
	}
}
