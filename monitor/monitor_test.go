package monitor

import (
	"fmt"
	"github.com.haa-criticals/watcher/watcher"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type testingErrorHandler struct {
	t             *testing.T
	failOnError   bool
	receivedError error
}

func (f *testingErrorHandler) OnHeartBeatError(err error, watcher *watcher.Peer) {
	f.receivedError = err
	if f.failOnError {
		f.t.Fatalf("error sending heart Beat to %s: %v", watcher.Address, err)
	}
}

func (f *testingErrorHandler) OnHealthCheckError(err error) {
	f.receivedError = err
	if f.failOnError {
		f.t.Fatalf("error sending health check: %v", err)
	}
}

func TestRegisterWatcher(t *testing.T) {
	t.Run("Should register watchers", func(t *testing.T) {
		m := New()
		assert.Len(t, m.peers, 0)
		m.RegisterWatcher(&watcher.Peer{})
		assert.Len(t, m.peers, 1)
	})
}

func TestHeartBeat(t *testing.T) {
	t.Run("Should send heart Beat to all watchers", func(t *testing.T) {
		endpoint1 := &mockEndpoint{}
		endpoint2 := &mockEndpoint{}
		endpoint1.start()
		endpoint2.start()

		start := time.Now()

		m := New(WithErrorHandler(&testingErrorHandler{t: t, failOnError: true}))

		m.peers = []*watcher.Peer{
			{Address: endpoint1.baseURL()},
			{Address: endpoint2.baseURL()},
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
		m := New()
		err := m.heartBeat()
		if assert.Error(t, err) {
			assert.Equal(t, "no peers registered", err.Error())
		}
	})

	t.Run("Should call error handler if there is an error sending heart Beat", func(t *testing.T) {
		handler := &testingErrorHandler{t: t}
		m := New(WithErrorHandler(handler))

		m.peers = []*watcher.Peer{
			{Address: "http://localhost:1234"},
		}

		err := m.heartBeat()
		assert.NoError(t, err)
		assert.Error(t, handler.receivedError)
	})

	t.Run("should send at least 5 heart Beats to all watchers in 6 millisecs", func(t *testing.T) {
		endpoint1 := &mockEndpoint{}
		endpoint2 := &mockEndpoint{}
		endpoint1.start()
		endpoint2.start()

		m := New(
			WithErrorHandler(&testingErrorHandler{t: t, failOnError: true}),
			WithHeartBeat(&defaultNotifier{}, 1*time.Millisecond),
		)

		m.peers = []*watcher.Peer{
			{Address: endpoint1.baseURL()},
			{Address: endpoint2.baseURL()},
		}

		go m.StartHeartBeating()
		time.Sleep(6 * time.Millisecond)
		m.Stop()
		assert.GreaterOrEqual(t, endpoint1.beatCount, 5)
		assert.GreaterOrEqual(t, endpoint2.beatCount, 5)

	})
}

func TestMonitor(t *testing.T) {
	t.Run("Should send at most 3 heart beats and 2 health checks in 10 millisecs", func(t *testing.T) {
		endpoint1 := &mockEndpoint{}
		endpoint1.start()

		m := New(
			WithErrorHandler(&testingErrorHandler{t: t, failOnError: true}),
			WithHeartBeat(&defaultNotifier{}, 5*time.Millisecond),
			WithHealthCheck(fmt.Sprintf("%s/healthz", endpoint1.baseURL()), 5*time.Millisecond, 3),
		)

		m.RegisterWatcher(&watcher.Peer{
			Address: fmt.Sprintf(endpoint1.baseURL()),
		})

		go m.StartHeartBeating()
		go m.StartHealthChecks()
		time.Sleep(10 * time.Millisecond)
		m.Stop()
		time.Sleep(10 * time.Millisecond)
		endpoint1.stop()
		assert.LessOrEqual(t, endpoint1.healthCheckCount, 2)
		assert.LessOrEqual(t, endpoint1.beatCount, 3)
	})

	t.Run("Only one heart beating should be running at a time", func(t *testing.T) {
		endpoint1 := &mockEndpoint{}
		endpoint1.start()

		m := New(
			WithErrorHandler(&testingErrorHandler{t: t, failOnError: true}),
			WithHeartBeat(&defaultNotifier{}, 5*time.Millisecond),
		)

		m.RegisterWatcher(&watcher.Peer{
			Address: fmt.Sprintf(endpoint1.baseURL()),
		})

		go m.StartHeartBeating()
		go m.StartHeartBeating()
		time.Sleep(15 * time.Millisecond)
		m.Stop()
		assert.LessOrEqual(t, endpoint1.beatCount, 4)
	})

	t.Run("Only one health check should be running at a time", func(t *testing.T) {
		endpoint1 := &mockEndpoint{}
		endpoint1.start()

		m := New(WithErrorHandler(&testingErrorHandler{t: t, failOnError: true}),
			WithHeartBeat(&defaultNotifier{}, 5*time.Millisecond))

		go m.StartHealthChecks()
		go m.StartHealthChecks()
		time.Sleep(15 * time.Millisecond)
		m.Stop()
		assert.LessOrEqual(t, endpoint1.healthCheckCount, 3)
	})

	t.Run("Should be unhealthy if the health check is not successful", func(t *testing.T) {
		endpoint1 := &mockEndpoint{}
		endpoint1.start()

		m := New(WithHealthCheck(fmt.Sprintf("%s/healthz", endpoint1.baseURL()), 1*time.Millisecond, 3))
		go m.StartHealthChecks()
		endpoint1.stop()
		time.Sleep(4 * time.Millisecond)
		m.Stop()
		assert.Falsef(t, m.IsHealthy(), "Should be unhealthy")
	})

	t.Run("Should be healthy if the health check is successful", func(t *testing.T) {
		endpoint1 := &mockEndpoint{}
		endpoint1.start()

		m := New(WithHealthCheck(fmt.Sprintf("%s/healthz", endpoint1.baseURL()), 1*time.Millisecond, 3))
		go m.StartHealthChecks()
		endpoint1.stop()
		time.Sleep(6 * time.Millisecond)
		m.Stop()
		assert.Falsef(t, m.IsHealthy(), "Should be unhealthy")

		endpoint1.start()
		m.healthChecker.endpoint = fmt.Sprintf("%s/healthz", endpoint1.baseURL())
		go m.StartHealthChecks()
		time.Sleep(10 * time.Millisecond)
		assert.True(t, m.IsHealthy(), "Should be healthy")

		m.Stop()
		endpoint1.stop()
	})
}
