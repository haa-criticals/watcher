package watcher

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestWatcher(t *testing.T) {
	t.Run("Should notify leader down after 3 seconds counting from last received beat", func(t *testing.T) {
		leaderDownTriggered := false
		w := &Watcher{
			lastReceivedBeat:       time.Now(),
			checkHeartBeatInterval: time.Second,
			maxLeaderAliveInterval: 2 * time.Second,
			OnLeaderDown: func(leader Info, lastReceivedBeat time.Time) {
				leaderDownTriggered = true
			},
		}

		go w.StartHeartBeatChecking()
		time.Sleep(3 * time.Second)
		assert.True(t, leaderDownTriggered)
	})
}
