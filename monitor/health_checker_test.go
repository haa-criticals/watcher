package monitor

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestHealthCheck(t *testing.T) {
	t.Run("Should return true if the endpoint is alive", func(t *testing.T) {
		endpoint1 := &mockEndpoint{}
		endpoint1.start()

		c := &healthChecker{}

		err := c.healthCheck(fmt.Sprintf("%s/healthz", endpoint1.baseURL()))
		assert.NoError(t, err)
		endpoint1.stop()
	})

	t.Run("Should return error the endpoint is dead", func(t *testing.T) {
		endpoint1 := &mockEndpoint{}
		endpoint1.start()

		c := &healthChecker{}
		healthEndpoint := fmt.Sprintf("%s/healthz", endpoint1.baseURL())

		err := c.healthCheck(healthEndpoint)
		assert.NoError(t, err)
		endpoint1.stop()
		err = c.healthCheck(healthEndpoint)
		assert.Error(t, err)
	})

	t.Run("Should send at least 5 health checks in 6 secs", func(t *testing.T) {
		endpoint1 := &mockEndpoint{}
		endpoint1.start()

		c := &healthChecker{
			endpoint: fmt.Sprintf("%s/healthz", endpoint1.baseURL()),
			interval: time.Second,
			maxFails: 3,
			done:     make(chan struct{}),
		}

		go c.Start()
		time.Sleep(6 * time.Second)
		c.Stop()
		assert.GreaterOrEqual(t, endpoint1.healthCheckCount, 5)

	})
}
