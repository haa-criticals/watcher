package monitor

import (
	"fmt"
	"log"
	"sync"
)

type Watcher struct {
	BaseURL string
}

type Monitor struct {
	notifier     Notifier
	watchers     []*Watcher
	ErrorHandler func(error, *Watcher)
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
		if m.ErrorHandler != nil {
			m.ErrorHandler(err, watcher)
		}
	}
}

func (m *Monitor) RegisterWatcher(watcher *Watcher) {
	m.watchers = append(m.watchers, watcher)
}

func New(notifier Notifier) *Monitor {
	return &Monitor{
		notifier: notifier,
	}
}
