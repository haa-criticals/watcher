package watcher

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestElection(t *testing.T) {
	t.Run("Should have at least one node to start election", func(t *testing.T) {
		var nodes []*Peer
		_, err := newElection(nodes, 0)
		if assert.Error(t, err) {
			assert.Equal(t, err.Error(), "no nodes to start election")
		}
	})
}
