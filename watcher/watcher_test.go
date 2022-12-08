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
			checkHeartBeatInterval: 1 * time.Second,
			maxLeaderAliveInterval: 2 * time.Second,
			OnLeaderDown: func(leader *NodeInfo, nodes []*NodeInfo, lastReceivedBeat time.Time) {
				leaderDownTriggered = true
			},
		}

		go w.StartHeartBeatChecking()
		time.Sleep(3 * time.Second)
		assert.True(t, leaderDownTriggered)
	})

	t.Run("Should not notify leader down if received beat in time", func(t *testing.T) {
		leaderDownTriggered := false
		w := &Watcher{
			lastReceivedBeat:       time.Now(),
			checkHeartBeatInterval: time.Second,
			maxLeaderAliveInterval: 2 * time.Second,
			OnLeaderDown: func(leader *NodeInfo, nodes []*NodeInfo, lastReceivedBeat time.Time) {
				leaderDownTriggered = true
			},
		}

		go w.StartHeartBeatChecking()
		time.Sleep(1 * time.Second)
		w.OnReceiveHeartBeat(time.Now())
		time.Sleep(1 * time.Second)
		w.OnReceiveHeartBeat(time.Now())
		assert.False(t, leaderDownTriggered)
	})

	t.Run("Should not start checking heart beat if already started", func(t *testing.T) {
		leaderDownTriggeredCount := 0
		w := &Watcher{
			lastReceivedBeat:       time.Now(),
			leader:                 &NodeInfo{},
			checkHeartBeatInterval: 2 * time.Second,
			maxLeaderAliveInterval: 2 * time.Second,
			OnLeaderDown: func(leader *NodeInfo, nodes []*NodeInfo, lastReceivedBeat time.Time) {
				leaderDownTriggeredCount = leaderDownTriggeredCount + 1
			},
		}

		go w.StartHeartBeatChecking()
		go w.StartHeartBeatChecking()
		time.Sleep(3 * time.Second)
		assert.Equal(t, 1, leaderDownTriggeredCount)
	})

	t.Run("Should not start checking heart beat if there is no leader", func(t *testing.T) {
		leaderDownTriggeredCount := 0
		w := &Watcher{
			lastReceivedBeat:       time.Now(),
			checkHeartBeatInterval: time.Second,
			maxLeaderAliveInterval: 2 * time.Second,
			OnLeaderDown: func(leader *NodeInfo, nodes []*NodeInfo, lastReceivedBeat time.Time) {
				leaderDownTriggeredCount = leaderDownTriggeredCount + 1
			},
		}

		go w.StartHeartBeatChecking()
		time.Sleep(3 * time.Second)
		assert.Equal(t, 0, leaderDownTriggeredCount)
	})

	t.Run("Should not notify leader down if stopped checking heart beat", func(t *testing.T) {
		leaderDownTriggered := false
		w := &Watcher{
			lastReceivedBeat:       time.Now(),
			leader:                 &NodeInfo{},
			checkHeartBeatInterval: 2 * time.Second,
			maxLeaderAliveInterval: 2 * time.Second,
			doneHeartBeatChecking:  make(chan struct{}),
			OnLeaderDown: func(leader *NodeInfo, nodes []*NodeInfo, lastReceivedBeat time.Time) {
				leaderDownTriggered = true
			},
		}

		go w.StartHeartBeatChecking()
		time.Sleep(time.Second)
		w.StopHeartBeatChecking()
		time.Sleep(2 * time.Second)
		assert.False(t, leaderDownTriggered)
	})

	t.Run("Should not notify leader down if min notification interval is not passed yet", func(t *testing.T) {
		leaderDownTriggeredCount := 0
		w := &Watcher{
			lastReceivedBeat:                  time.Now(),
			leader:                            &NodeInfo{},
			checkHeartBeatInterval:            1 * time.Second,
			maxLeaderAliveInterval:            2 * time.Second,
			minLeaderDownNotificationInterval: 3 * time.Second,
			OnLeaderDown: func(leader *NodeInfo, nodes []*NodeInfo, lastReceivedBeat time.Time) {
				leaderDownTriggeredCount++
			},
		}

		go w.StartHeartBeatChecking()
		time.Sleep(4 * time.Second)
		assert.Equal(t, 1, leaderDownTriggeredCount)
	})

	t.Run("Should create an election if leader is down", func(t *testing.T) {
		w := &Watcher{
			lastReceivedBeat:                  time.Now(),
			leader:                            &NodeInfo{},
			checkHeartBeatInterval:            1 * time.Second,
			maxLeaderAliveInterval:            2 * time.Second,
			minLeaderDownNotificationInterval: 3 * time.Second,
		}

		go w.StartHeartBeatChecking()
		time.Sleep(4 * time.Second)
		assert.NotNil(t, w.election)
	})

	t.Run("Should start an election if leader is down", func(t *testing.T) {
		w := &Watcher{
			lastReceivedBeat:                  time.Now(),
			leader:                            &NodeInfo{},
			checkHeartBeatInterval:            1 * time.Second,
			maxLeaderAliveInterval:            2 * time.Second,
			minLeaderDownNotificationInterval: 3 * time.Second,
			nodes: []*NodeInfo{
				{"192.168.0.10", 1},
				{"191.168.0.11", 2},
			},
		}

		go w.StartHeartBeatChecking()
		time.Sleep(4 * time.Second)
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
