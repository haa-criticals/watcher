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

func (f *testingErrorHandler) OnHeartBeatError(err error, watcher *watcher.NodeInfo) {
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
		assert.Len(t, m.watchers, 0)
		m.RegisterWatcher(&watcher.NodeInfo{})
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

		m := New(WithErrorHandler(&testingErrorHandler{t: t, failOnError: true}))

		m.watchers = []*watcher.NodeInfo{
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
			assert.Equal(t, "no watchers registered", err.Error())
		}
	})

	t.Run("Should call error handler if there is an error sending heart Beat", func(t *testing.T) {
		handler := &testingErrorHandler{t: t}
		m := New(WithErrorHandler(handler))

		m.watchers = []*watcher.NodeInfo{
			{Address: "http://localhost:1234"},
		}

		err := m.heartBeat()
		assert.NoError(t, err)
		assert.Error(t, handler.receivedError)
	})

	t.Run("should send at least 5 heart Beats to all watchers in 6 secs", func(t *testing.T) {
		endpoint1 := &mockEndpoint{}
		endpoint2 := &mockEndpoint{}
		endpoint1.start()
		endpoint2.start()

		m := New(
			WithErrorHandler(&testingErrorHandler{t: t, failOnError: true}),
			WithHeartBeat(&defaultNotifier{}, time.Second),
		)

		m.watchers = []*watcher.NodeInfo{
			{Address: endpoint1.baseURL()},
			{Address: endpoint2.baseURL()},
		}

		go m.StartHeartBeating()
		time.Sleep(6 * time.Second)
		m.Stop()
		assert.GreaterOrEqual(t, endpoint1.beatCount, 5)
		assert.GreaterOrEqual(t, endpoint2.beatCount, 5)

	})
}

func TestMonitor(t *testing.T) {
	t.Run("Should send at most 2 heart beats and 2 health checks in 2 secs", func(t *testing.T) {
		endpoint1 := &mockEndpoint{}
		endpoint1.start()

		m := New(
			WithErrorHandler(&testingErrorHandler{t: t, failOnError: true}),
			WithHeartBeat(&defaultNotifier{}, time.Second),
			WithHealthCheck(fmt.Sprintf("%s/healthz", endpoint1.baseURL()), time.Second, 3),
		)

		m.RegisterWatcher(&watcher.NodeInfo{
			Address: fmt.Sprintf(endpoint1.baseURL()),
		})

		go m.StartHeartBeating()
		go m.StartHealthChecks()
		time.Sleep(2 * time.Second)
		m.Stop()
		time.Sleep(3 * time.Second)
		endpoint1.stop()
		assert.LessOrEqual(t, endpoint1.healthCheckCount, 2)
		assert.LessOrEqual(t, endpoint1.beatCount, 2)
	})

	t.Run("Only one heart beating should be running at a time", func(t *testing.T) {
		endpoint1 := &mockEndpoint{}
		endpoint1.start()

		m := New(
			WithErrorHandler(&testingErrorHandler{t: t, failOnError: true}),
			WithHeartBeat(&defaultNotifier{}, time.Second),
		)

		m.RegisterWatcher(&watcher.NodeInfo{
			Address: fmt.Sprintf(endpoint1.baseURL()),
		})

		go m.StartHeartBeating()
		go m.StartHeartBeating()
		time.Sleep(3 * time.Second)
		m.Stop()
		assert.LessOrEqual(t, endpoint1.beatCount, 3)
	})

	t.Run("Only one health check should be running at a time", func(t *testing.T) {
		endpoint1 := &mockEndpoint{}
		endpoint1.start()

		m := New(WithErrorHandler(&testingErrorHandler{t: t, failOnError: true}))

		go m.StartHealthChecks()
		go m.StartHealthChecks()
		time.Sleep(3 * time.Second)
		m.Stop()
		assert.LessOrEqual(t, endpoint1.healthCheckCount, 3)
	})

	t.Run("Should be unhealthy if the health check is not successful", func(t *testing.T) {
		endpoint1 := &mockEndpoint{}
		endpoint1.start()

		m := New(WithHealthCheck(fmt.Sprintf("%s/healthz", endpoint1.baseURL()), time.Second, 3))
		go m.StartHealthChecks()
		endpoint1.stop()
		time.Sleep(4 * time.Second)
		m.Stop()
		assert.Falsef(t, m.IsHealthy(), "Should be unhealthy")
	})

	t.Run("Should be healthy if the health check is successful", func(t *testing.T) {
		endpoint1 := &mockEndpoint{}
		endpoint1.start()

		m := New(WithHealthCheck(fmt.Sprintf("%s/healthz", endpoint1.baseURL()), time.Second, 3))
		go m.StartHealthChecks()
		endpoint1.stop()
		time.Sleep(4 * time.Second)
		m.Stop()
		assert.Falsef(t, m.IsHealthy(), "Should be unhealthy")

		endpoint1.start()
		m.healthChecker.endpoint = fmt.Sprintf("%s/healthz", endpoint1.baseURL())
		go m.StartHealthChecks()
		time.Sleep(2 * time.Second)
		assert.True(t, m.IsHealthy(), "Should be healthy")

		m.Stop()
		endpoint1.stop()
	})
}
