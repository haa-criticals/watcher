package watcher

import (
	"github.com/google/uuid"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestElection(t *testing.T) {
	t.Run("Should have at least one node to start election", func(t *testing.T) {
		var nodes []*NodeInfo
		e := New(nodes)
		err := e.Start()
		if assert.Error(t, err) {
			assert.Equal(t, err.Error(), "no nodes to start election")
		}
	})

	t.Run("Should have received a accept election from all nodes before start the election", func(t *testing.T) {
		nodes := []*NodeInfo{
			{ID: uuid.UUID{}, BaseURL: "locahost:50051"},
			{ID: uuid.UUID{}, BaseURL: "locahost:50052"},
			{ID: uuid.UUID{}, BaseURL: "locahost:50053"},
		}
		e := New(nodes)
		err := e.Start()
		if assert.Error(t, err) {
			assert.Equal(t, err.Error(), "not all nodes accepted election yet")
		}
	})
}
