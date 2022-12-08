package watcher

import (
	"context"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

type mockWatcherClient struct {
	fRequestRegister func(ctx context.Context, address, key string) (*RegisterResponse, error)
	fAckNode         func(ctx context.Context, address, key string, node *NodeInfo) (*NodeInfo, error)
}

func (m *mockWatcherClient) RequestRegister(ctx context.Context, address, key string) (*RegisterResponse, error) {
	return m.fRequestRegister(ctx, address, key)
}

func (m *mockWatcherClient) AckNode(ctx context.Context, address, key string, node *NodeInfo) (*NodeInfo, error) {
	return m.fAckNode(ctx, address, key, node)
}

func TestWatcher(t *testing.T) {
	t.Run("Should notify leader down after 3 seconds counting from last received beat", func(t *testing.T) {
		leaderDownTriggered := false
		w := &Watcher{
			lastReceivedBeat:       time.Now(),
			leader:                 &NodeInfo{},
			checkHeartBeatInterval: 5 * time.Millisecond,
			maxLeaderAliveInterval: 10 * time.Millisecond,
			OnLeaderDown: func(leader *NodeInfo, nodes []*NodeInfo, lastReceivedBeat time.Time) {
				leaderDownTriggered = true
			},
		}

		go w.StartHeartBeatChecking()
		time.Sleep(15 * time.Millisecond)
		assert.True(t, leaderDownTriggered)
	})

	t.Run("Should not notify leader down if received beat in time", func(t *testing.T) {
		leaderDownTriggered := false
		w := &Watcher{
			lastReceivedBeat:       time.Now(),
			checkHeartBeatInterval: 5 * time.Millisecond,
			maxLeaderAliveInterval: 10 * time.Millisecond,
			OnLeaderDown: func(leader *NodeInfo, nodes []*NodeInfo, lastReceivedBeat time.Time) {
				leaderDownTriggered = true
			},
		}

		go w.StartHeartBeatChecking()
		time.Sleep(5 * time.Millisecond)
		w.OnReceiveHeartBeat(time.Now())
		time.Sleep(5 * time.Millisecond)
		w.OnReceiveHeartBeat(time.Now())
		assert.False(t, leaderDownTriggered)
	})

	t.Run("Should not start checking heart beat if already started", func(t *testing.T) {
		leaderDownTriggeredCount := 0
		w := &Watcher{
			lastReceivedBeat:       time.Now(),
			leader:                 &NodeInfo{},
			checkHeartBeatInterval: 10 * time.Millisecond,
			maxLeaderAliveInterval: 10 * time.Millisecond,
			OnLeaderDown: func(leader *NodeInfo, nodes []*NodeInfo, lastReceivedBeat time.Time) {
				leaderDownTriggeredCount = leaderDownTriggeredCount + 1
			},
		}

		go w.StartHeartBeatChecking()
		go w.StartHeartBeatChecking()
		time.Sleep(15 * time.Millisecond)
		assert.Equal(t, 1, leaderDownTriggeredCount)
	})

	t.Run("Should not start checking heart beat if there is no leader", func(t *testing.T) {
		leaderDownTriggeredCount := 0
		w := &Watcher{
			lastReceivedBeat:       time.Now(),
			checkHeartBeatInterval: 5 * time.Millisecond,
			maxLeaderAliveInterval: 10 * time.Millisecond,
			OnLeaderDown: func(leader *NodeInfo, nodes []*NodeInfo, lastReceivedBeat time.Time) {
				leaderDownTriggeredCount = leaderDownTriggeredCount + 1
			},
		}

		go w.StartHeartBeatChecking()
		time.Sleep(15 * time.Millisecond)
		assert.Equal(t, 0, leaderDownTriggeredCount)
	})

	t.Run("Should not notify leader down if stopped checking heart beat", func(t *testing.T) {
		leaderDownTriggered := false
		w := &Watcher{
			lastReceivedBeat:       time.Now(),
			leader:                 &NodeInfo{},
			checkHeartBeatInterval: 10 * time.Millisecond,
			maxLeaderAliveInterval: 10 * time.Millisecond,
			doneHeartBeatChecking:  make(chan struct{}),
			OnLeaderDown: func(leader *NodeInfo, nodes []*NodeInfo, lastReceivedBeat time.Time) {
				leaderDownTriggered = true
			},
		}

		go w.StartHeartBeatChecking()
		time.Sleep(5 * time.Millisecond)
		w.StopHeartBeatChecking()
		time.Sleep(10 * time.Millisecond)
		assert.False(t, leaderDownTriggered)
	})

	t.Run("Should not notify leader down if min notification interval is not passed yet", func(t *testing.T) {
		leaderDownTriggeredCount := 0
		w := &Watcher{
			lastReceivedBeat:                  time.Now(),
			leader:                            &NodeInfo{},
			checkHeartBeatInterval:            5 * time.Millisecond,
			maxLeaderAliveInterval:            10 * time.Millisecond,
			minLeaderDownNotificationInterval: 15 * time.Millisecond,
			OnLeaderDown: func(leader *NodeInfo, nodes []*NodeInfo, lastReceivedBeat time.Time) {
				leaderDownTriggeredCount++
			},
		}

		go w.StartHeartBeatChecking()
		time.Sleep(20 * time.Millisecond)
		assert.Equal(t, 1, leaderDownTriggeredCount)
	})

	t.Run("Should create an election if leader is down", func(t *testing.T) {
		w := &Watcher{
			lastReceivedBeat:                  time.Now(),
			leader:                            &NodeInfo{},
			checkHeartBeatInterval:            5 * time.Millisecond,
			maxLeaderAliveInterval:            10 * time.Millisecond,
			minLeaderDownNotificationInterval: 15 * time.Millisecond,
		}

		go w.StartHeartBeatChecking()
		time.Sleep(20 * time.Millisecond)
		assert.NotNil(t, w.election)
	})

	t.Run("Should start an election if leader is down", func(t *testing.T) {
		w := &Watcher{
			lastReceivedBeat:                  time.Now(),
			leader:                            &NodeInfo{},
			checkHeartBeatInterval:            5 * time.Millisecond,
			maxLeaderAliveInterval:            10 * time.Millisecond,
			minLeaderDownNotificationInterval: 15 * time.Millisecond,
			nodes: []*NodeInfo{
				{"192.168.0.10", 1},
				{"191.168.0.11", 2},
			},
		}

		go w.StartHeartBeatChecking()
		time.Sleep(20 * time.Millisecond)
		assert.NotNil(t, w.election)
		assert.False(t, w.election.startedAt.IsZero())
	})

	t.Run("Should register nodes", func(t *testing.T) {
		w := &Watcher{
			registrationKey: "key",
			registerLocker:  &sync.Mutex{},
		}
		_, err := w.RegisterNode(&NodeInfo{Address: "http://localhost:8080"}, "key")
		assert.NoError(t, err)
		_, err = w.RegisterNode(&NodeInfo{Address: "http://localhost:8081"}, "key")
		assert.NoError(t, err)
		assert.Equal(t, 2, len(w.nodes))
	})

	t.Run("Node Register should fail if the key does not matches", func(t *testing.T) {
		w := &Watcher{
			registrationKey: "key",
		}
		_, err := w.RegisterNode(&NodeInfo{Address: "http://localhost:8080"}, "key1")
		assert.Error(t, err)
		assert.Equal(t, 0, len(w.nodes))
	})

	t.Run("Should return all registered nodes, on new node registration", func(t *testing.T) {
		w := &Watcher{
			registrationKey: "key",
			registerLocker:  &sync.Mutex{},
		}
		nodes, err := w.RegisterNode(&NodeInfo{Address: "http://localhost:8080"}, "key")
		assert.NoError(t, err)
		assert.Equal(t, 1, len(nodes))
		nodes, err = w.RegisterNode(&NodeInfo{Address: "http://localhost:8081"}, "key")
		assert.NoError(t, err)
		assert.Equal(t, 2, len(nodes))
		nodes, err = w.RegisterNode(&NodeInfo{Address: "http://localhost:8082"}, "key")
		assert.NoError(t, err)
		assert.Equal(t, 3, len(nodes))
	})

	t.Run("The leader should be set after requesting registration", func(t *testing.T) {
		w := &Watcher{
			registrationKey: "key",
			registerLocker:  &sync.Mutex{},
			client: &mockWatcherClient{
				fRequestRegister: func(ctx context.Context, address, key string) (*RegisterResponse, error) {
					return &RegisterResponse{
						Success: true,
						Nodes:   []*NodeInfo{{Address: "http://localhost:8081"}},
					}, nil
				},
			},
		}
		err := w.RequestRegister("localhost:8080", "key")
		assert.NoError(t, err)
		assert.NotNil(t, w.leader)
	})
}
