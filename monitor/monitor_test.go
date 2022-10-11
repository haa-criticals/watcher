package monitor

import (
	"fmt"
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
		m.HeartBeatErrorHandler = func(err error, watcher *Watcher) {
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
		m.HeartBeatErrorHandler = func(err error, watcher *Watcher) {
			receivedError = err
		}

		m.watchers = []*Watcher{
			{BaseURL: "http://localhost:1234"},
		}

		err := m.heartBeat()
		assert.NoError(t, err)
		assert.NotNil(t, receivedError)
	})

	t.Run("should send at least 5 heart Beats to all watchers in 6 secs", func(t *testing.T) {
		endpoint1 := &mockEndpoint{}
		endpoint2 := &mockEndpoint{}
		endpoint1.start()
		endpoint2.start()

		m := New(&defaultNotifier{})
		m.HeartBeatErrorHandler = func(err error, watcher *Watcher) {
			t.Fatalf("error sending heart Beat to %s: %v", watcher.BaseURL, err)
		}

		m.watchers = []*Watcher{
			{BaseURL: endpoint1.baseURL()},
			{BaseURL: endpoint2.baseURL()},
		}

		go m.StartHeartBeating(time.Second)
		time.Sleep(6 * time.Second)
		m.Stop()
		assert.GreaterOrEqual(t, endpoint1.beatCount, 5)
		assert.GreaterOrEqual(t, endpoint2.beatCount, 5)

	})
}

func TestHealthCheck(t *testing.T) {
	t.Run("Should return true if the endpoint is alive", func(t *testing.T) {
		endpoint1 := &mockEndpoint{}
		endpoint1.start()

		m := New(&defaultNotifier{})

		err := m.healthCheck(fmt.Sprintf("%s/healthz", endpoint1.baseURL()))
		assert.NoError(t, err)
		endpoint1.stop()
	})

	t.Run("Should return error the endpoint is dead", func(t *testing.T) {
		endpoint1 := &mockEndpoint{}
		endpoint1.start()

		m := New(&defaultNotifier{})
		healthEndpoint := fmt.Sprintf("%s/healthz", endpoint1.baseURL())

		err := m.healthCheck(healthEndpoint)
		assert.NoError(t, err)
		endpoint1.stop()
		err = m.healthCheck(healthEndpoint)
		assert.Error(t, err)
	})

	t.Run("Should send at least 5 health checks in 6 secs", func(t *testing.T) {
		endpoint1 := &mockEndpoint{}
		endpoint1.start()

		m := New(&defaultNotifier{})

		go m.StartHealthChecks(time.Second, fmt.Sprintf("%s/healthz", endpoint1.baseURL()))
		time.Sleep(6 * time.Second)
		m.Stop()
		assert.GreaterOrEqual(t, endpoint1.healthCheckCount, 5)

	})
}

func TestMonitor(t *testing.T) {
	t.Run("Should send at most 2 heart beats and 2 health checks in 2 secs", func(t *testing.T) {
		endpoint1 := &mockEndpoint{}
		endpoint1.start()

		m := New(&defaultNotifier{})
		m.HeartBeatErrorHandler = func(err error, watcher *Watcher) {
			t.Fatalf("error sending heart Beat to %s: %v", watcher.BaseURL, err)
		}

		m.RegisterWatcher(&Watcher{
			BaseURL: fmt.Sprintf(endpoint1.baseURL()),
		})

		go m.StartHeartBeating(time.Second)
		go m.StartHealthChecks(time.Second, fmt.Sprintf("%s/healthz", endpoint1.baseURL()))
		time.Sleep(2 * time.Second)
		m.Stop()
		time.Sleep(3 * time.Second)
		assert.LessOrEqual(t, endpoint1.healthCheckCount, 2)
		assert.LessOrEqual(t, endpoint1.beatCount, 2)

	})
}
