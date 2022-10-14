package monitor

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type healthChecker struct {
	endpoint     string
	done         chan struct{}
	interval     time.Duration
	active       bool
	fails        int
	maxFails     int
	locker       sync.Mutex
	errorHandler func(error)
}

func (c *healthChecker) Start() {
	if c.endpoint == "" {
		log.Printf("no health check endpoint provided")
		return
	}
	c.locker.Lock()
	if c.active {
		c.locker.Unlock()
		return
	}
	c.active = true
	c.locker.Unlock()

	t := time.NewTicker(c.interval)
	for {
		select {
		case <-t.C:
			err := c.healthCheck(c.endpoint)
			if err != nil {
				c.fails++
				c.errorHandler(err)
			} else {
				c.fails = 0
			}
		case <-c.done:
			t.Stop()
			return
		}
	}
}

func (c *healthChecker) healthCheck(endpoint string) error {
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

func (c *healthChecker) Stop() {
	if c.active {
		c.done <- struct{}{}
		c.active = false
	}
}
