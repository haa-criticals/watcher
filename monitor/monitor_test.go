package monitor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRegisterWatcher(t *testing.T) {
	t.Run("Should register watchers", func(t *testing.T) {
		m := New(&defaultNotifier{})
		assert.Len(t, m.watchers, 0)
		m.RegisterWatcher(&Watcher{})
		assert.Len(t, m.watchers, 1)
	})
}

func TestHeartBeat(t *testing.T) {
	t.Run("Should send heart Beat to all watchers", func(t *testing.T) {
		endpoint1 := &mockEndpoint{}
		endpoint2 := &mockEndpoint{}
		endpoint1.start()
		endpoint2.start()

		start := time.Now()

		m := New(&defaultNotifier{})
		m.ErrorHandler = func(err error, watcher *Watcher) {
			t.Fatalf("error sending heart Beat to %s: %v", watcher.BaseURL, err)
		}

		m.watchers = []*Watcher{
			{BaseURL: endpoint1.baseURL()},
			{BaseURL: endpoint2.baseURL()},
		}

		err := m.heartBeat()

		assert.NoError(t, err)
		assert.NotNil(t, endpoint1.lastBeat)
		assert.NotNil(t, endpoint2.lastBeat)
		assert.True(t, endpoint1.lastBeat.After(start), "last Beat should be after start")
		assert.True(t, endpoint2.lastBeat.After(start), "last Beat should be after start")
		endpoint1.stop()
		endpoint2.stop()
	})

	t.Run("Should return error if there is no registered watchers", func(t *testing.T) {
		m := New(&defaultNotifier{})
		err := m.heartBeat()
		if assert.Error(t, err) {
			assert.Equal(t, "no watchers registered", err.Error())
		}
	})

	t.Run("Should call error handler if there is an error sending heart Beat", func(t *testing.T) {
		m := New(&defaultNotifier{})
		var receivedError error
		m.ErrorHandler = func(err error, watcher *Watcher) {
			receivedError = err
		}

		m.watchers = []*Watcher{
			{BaseURL: "http://localhost:1234"},
		}

		err := m.heartBeat()
		assert.NoError(t, err)
		assert.NotNil(t, receivedError)
	})

	t.Run("should send 5 heart Beats to all watchers in 5 secs", func(t *testing.T) {
		endpoint1 := &mockEndpoint{}
		endpoint2 := &mockEndpoint{}
		endpoint1.start()
		endpoint2.start()

		m := New(&defaultNotifier{})
		m.ErrorHandler = func(err error, watcher *Watcher) {
			t.Fatalf("error sending heart Beat to %s: %v", watcher.BaseURL, err)
		}

		m.watchers = []*Watcher{
			{BaseURL: endpoint1.baseURL()},
			{BaseURL: endpoint2.baseURL()},
		}

		go m.StartHeartBeat(time.Second)
		time.Sleep(5 * time.Second)
		m.StopHeartBeat()
		assert.Equal(t, 5, endpoint1.beatCount)
		assert.Equal(t, 5, endpoint2.beatCount)
		endpoint1.stop()
		endpoint2.stop()

	})
}
