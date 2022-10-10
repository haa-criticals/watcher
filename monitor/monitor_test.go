package monitor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRegisterWatcher(t *testing.T) {
	t.Run("Should register watchers", func(t *testing.T) {
		m := New()
		assert.Len(t, m.watchers, 0)
		m.RegisterWatcher(&Watcher{})
		assert.Len(t, m.watchers, 1)
	})
}

func TestHeartBeat(t *testing.T) {
	t.Run("Should send heart beat to all watchers", func(t *testing.T) {
		endpoint1 := &mockEndpoint{}
		endpoint2 := &mockEndpoint{}
		endpoint1.start()
		endpoint2.start()

		start := time.Now()

		m := New()
		m.ErrorHandler = func(err error, watcher *Watcher) {
			t.Fatalf("error sending heart beat to %s: %v", watcher.BaseURL, err)
		}

		m.watchers = []*Watcher{
			{BaseURL: endpoint1.baseURL()},
			{BaseURL: endpoint2.baseURL()},
		}

		err := m.heartBeat()

		assert.NoError(t, err)
		assert.NotNil(t, endpoint1.lastBeat)
		assert.NotNil(t, endpoint2.lastBeat)
		assert.True(t, endpoint1.lastBeat.After(start), "last beat should be after start")
		assert.True(t, endpoint2.lastBeat.After(start), "last beat should be after start")
		endpoint1.stop()
		endpoint2.stop()
	})

	t.Run("Should return error if there is no registered watchers", func(t *testing.T) {
		m := New()
		err := m.heartBeat()
		if assert.Error(t, err) {
			assert.Equal(t, "no watchers registered", err.Error())
		}
	})

	t.Run("Should call error handler if there is an error sending heart beat", func(t *testing.T) {
		m := New()
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
}
