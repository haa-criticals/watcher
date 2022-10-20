package watcher

import (
	"github.com/google/uuid"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestElection(t *testing.T) {
	t.Run("Should have at least one node to start election", func(t *testing.T) {
		var nodes []*NodeInfo
		e := NewElection(nodes)
		err := e.Start()
		if assert.Error(t, err) {
			assert.Equal(t, err.Error(), "no nodes to start election")
		}
	})

	t.Run("Should have received a accept election from all nodes before start the election", func(t *testing.T) {
		nodes := []*NodeInfo{
			{ID: uuid.UUID{}, Address: "locahost:50051"},
			{ID: uuid.UUID{}, Address: "locahost:50052"},
			{ID: uuid.UUID{}, Address: "locahost:50053"},
		}
		e := NewElection(nodes)
		err := e.Start()
		if assert.Error(t, err) {
			assert.Equal(t, err.Error(), "not all nodes accepted election yet")
		}
	})

	t.Run("Should start election", func(t *testing.T) {
		nodes := []*NodeInfo{
			{ID: uuid.UUID{}, Address: "locahost:50051", electionState: accepted},
			{ID: uuid.UUID{}, Address: "locahost:50052", electionState: accepted},
			{ID: uuid.UUID{}, Address: "locahost:50053", electionState: accepted},
		}
		e := NewElection(nodes)
		started := false
		e.OnStartElection = func(nodes []*NodeInfo) {
			started = true
		}
		err := e.Start()
		assert.NoError(t, err)
		assert.True(t, started)
	})

	t.Run("Should wait until all nodes has registered to elect a new leader", func(t *testing.T) {
		nodes := []*NodeInfo{
			{ID: uuid.UUID{}, Address: "locahost:50051", electionState: accepted},
			{ID: uuid.UUID{}, Address: "locahost:50052", electionState: accepted},
			{ID: uuid.UUID{}, Address: "locahost:50053", electionState: accepted},
		}
		e := NewElection(nodes)
		go e.WaitRegistration()
		assert.Nil(t, e.newLeader)
		time.Sleep(1 * time.Second)
		e.ReceivePriority(nodes[0].ID, 2)
		e.ReceivePriority(nodes[1].ID, 2)
		assert.Nil(t, e.newLeader)
		time.Sleep(1 * time.Second)
		e.ReceivePriority(nodes[2].ID, 1)
		time.Sleep(2 * time.Second)
		assert.NotNil(t, e.newLeader)
	})

	t.Run("Should elect the node with the highest priority", func(t *testing.T) {
		nodes := []*NodeInfo{
			{ID: uuid.UUID{}, Address: "locahost:50051", electionState: accepted},
			{ID: uuid.UUID{}, Address: "locahost:50052", electionState: accepted},
			{ID: uuid.UUID{}, Address: "locahost:50053", electionState: accepted},
		}
		e := NewElection(nodes)
		go e.WaitRegistration()
		assert.Nil(t, e.newLeader)
		time.Sleep(1 * time.Second)
		e.ReceivePriority(nodes[0].ID, 2)
		e.ReceivePriority(nodes[1].ID, 1)
		assert.Nil(t, e.newLeader)
		time.Sleep(1 * time.Second)
		e.ReceivePriority(nodes[2].ID, 1)
		time.Sleep(2 * time.Second)
		assert.NotNil(t, e.newLeader)
		assert.Equal(t, e.newLeader, nodes[0])
	})

	t.Run("Should wait until all nodes has voted to elect a new leader", func(t *testing.T) {
		nodes := []*NodeInfo{
			{ID: uuid.UUID{}, Address: "locahost:50051", electionState: accepted},
			{ID: uuid.UUID{}, Address: "locahost:50052", electionState: accepted},
			{ID: uuid.UUID{}, Address: "locahost:50053", electionState: accepted},
		}
		e := NewElection(nodes)
		newLeaderElect := false
		e.OnNewLeaderElect = func(node *NodeInfo) {
			newLeaderElect = true
		}
		go e.WaitVotes()
		assert.Nil(t, e.newLeader)
		assert.False(t, newLeaderElect)
		time.Sleep(1 * time.Second)
		e.ReceiveVote(nodes[0].ID, nodes[1].ID)
		e.ReceiveVote(nodes[1].ID, nodes[1].ID)
		assert.False(t, newLeaderElect)
		assert.Nil(t, e.newLeader)
		time.Sleep(1 * time.Second)
		e.ReceiveVote(nodes[2].ID, nodes[1].ID)
		time.Sleep(2 * time.Second)
		assert.True(t, newLeaderElect)
		assert.NotNil(t, e.newLeader)
	})

}
