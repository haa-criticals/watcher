package app

import (
	"context"
	"fmt"
	"github.com.haa-criticals/watcher/provisioner"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	igrpc "github.com.haa-criticals/watcher/app/grpc"
	"github.com.haa-criticals/watcher/monitor"
	"github.com.haa-criticals/watcher/watcher"
)

type mockProvider struct {
	fCreate  func(ctx context.Context) error
	fDestroy func(ctx context.Context) error
}

func (m *mockProvider) Create(ctx context.Context) error {
	return m.fCreate(ctx)
}

func (m *mockProvider) Destroy(ctx context.Context) error {
	return m.fDestroy(ctx)
}

var (
	defaultWc = watcher.Config{
		MaxDelayForElection:    50,
		HeartBeatCheckInterval: 15 * time.Millisecond,
	}
)

func startNewNode(t *testing.T, address string, config *Config, provider provisioner.Provider, wc watcher.Config, monitorOpts ...monitor.Option) *App {
	t.Helper()
	node := New(
		watcher.New(igrpc.NewWatchClient(), wc),
		monitor.New(monitorOpts...),
		provisioner.WithProvider(provider),
		config,
	)

	go func() {
		err := node.Start()
		if err != nil {
			t.Errorf("failed to start node %s:  %v", address, err)
		}
	}()
	return node
}

func discoverLeader(t *testing.T, nodes ...*App) (*App, *App, *App) {
	t.Helper()
	if nodes[0].IsLeader() {
		return nodes[0], nodes[1], nodes[2]
	}
	if nodes[1].IsLeader() {
		return nodes[1], nodes[0], nodes[2]
	}
	return nodes[2], nodes[0], nodes[1]
}

func TestAppStart(t *testing.T) {
	t.Run("Should call the provisioner create when the app starts", func(t *testing.T) {

		createCalled := false
		provider := &mockProvider{
			fCreate: func(ctx context.Context) error {
				createCalled = true
				return nil
			},
		}

		monitorHB := monitor.WithHeartBeat(igrpc.NewWatchClient(), 1*time.Millisecond)

		node1 := startNewNode(t, ":50051", &Config{
			Address: ":50051",
			Peers:   []string{":50052", ":50053"},
		}, provider, defaultWc, monitorHB)

		node2 := startNewNode(t, ":50052", &Config{
			Address: ":50052",
			Peers:   []string{":50051", ":50053"},
		}, provider, defaultWc, monitorHB)

		node3 := startNewNode(t, ":50053", &Config{
			Address: ":50053",
			Peers:   []string{":50051", ":50052"},
		}, provider, defaultWc, monitorHB)

		time.Sleep(100 * time.Millisecond)
		assert.True(t, createCalled)
		node1.Stop()
		node2.Stop()
		node3.Stop()

	})

	t.Run("Should start health check when the app starts", func(t *testing.T) {
		healthCalled := false
		mux := http.NewServeMux()
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			healthCalled = true
		})

		server := httptest.NewUnstartedServer(mux)
		provider := &mockProvider{
			fCreate: func(ctx context.Context) error {
				server.Start()
				return nil
			},
		}

		monitorHB := monitor.WithHeartBeat(igrpc.NewWatchClient(), 5*time.Millisecond)
		monitorHC := monitor.WithHealthCheck(fmt.Sprintf("http://%s%s", server.Listener.Addr(), "/health"), 1*time.Millisecond, 3)

		node1 := startNewNode(t, ":50051", &Config{
			Address: ":50051",
			Peers:   []string{":50052", ":50053"},
		}, provider, defaultWc, monitorHC, monitorHB)

		node2 := startNewNode(t, ":50052", &Config{
			Address: ":50052",
			Peers:   []string{":50051", ":50053"},
		}, provider, defaultWc, monitorHC, monitorHB)

		node3 := startNewNode(t, ":50053", &Config{
			Address: ":50053",
			Peers:   []string{":50051", ":50052"},
		}, provider, defaultWc, monitorHC, monitorHB)

		time.Sleep(100 * time.Millisecond)
		assert.True(t, healthCalled)
		assert.True(t, node1.monitor.IsHealthy() || node2.monitor.IsHealthy() || node3.monitor.IsHealthy())
		node1.Stop()
		node2.Stop()
		node3.Stop()
		server.Close()
	})
}

func TestApp(t *testing.T) {
	t.Run("A new leader should be elected when the current leader is down", func(t *testing.T) {

		provider := &mockProvider{
			fCreate: func(ctx context.Context) error {
				return nil
			},
		}

		monitorHB := monitor.WithHeartBeat(igrpc.NewWatchClient(), 5*time.Millisecond)

		node1 := startNewNode(t, ":50051", &Config{
			Address: ":50051",
			Peers:   []string{":50052", ":50053"},
		}, provider, watcher.Config{
			MaxDelayForElection:    50,
			HeartBeatCheckInterval: 15 * time.Millisecond,
		}, monitorHB)

		node2 := startNewNode(t, ":50052", &Config{
			Address: ":50052",
			Peers:   []string{":50051", ":50053"},
		}, provider, defaultWc, monitorHB)

		node3 := startNewNode(t, ":50053", &Config{
			Address: ":50053",
			Peers:   []string{":50051", ":50052"},
		}, provider, defaultWc, monitorHB)

		time.Sleep(200 * time.Millisecond)

		leader, n1, n2 := discoverLeader(t, node1, node2, node3)

		assert.True(t, leader.IsLeader())
		assert.False(t, n1.IsLeader())
		assert.False(t, n2.IsLeader())
		leader.Stop()
		time.Sleep(500 * time.Millisecond)
		assert.True(t, n1.IsLeader() || n2.IsLeader())
		n1.Stop()
		n2.Stop()
	})

	t.Run("A elected node should start to monitor", func(t *testing.T) {
		provider := &mockProvider{
			fCreate: func(ctx context.Context) error {
				return nil
			},
		}

		monitorHB := monitor.WithHeartBeat(igrpc.NewWatchClient(), 5*time.Millisecond)

		node1 := startNewNode(t, ":50051", &Config{
			Address: ":50051",
			Peers:   []string{":50052", ":50053"},
		}, provider, watcher.Config{
			MaxDelayForElection:    50,
			HeartBeatCheckInterval: 15 * time.Millisecond,
		}, monitorHB)

		node2 := startNewNode(t, ":50052", &Config{
			Address: ":50052",
			Peers:   []string{":50051", ":50053"},
		}, provider, defaultWc, monitorHB)

		node3 := startNewNode(t, ":50053", &Config{
			Address: ":50053",
			Peers:   []string{":50051", ":50052"},
		}, provider, defaultWc, monitorHB)

		time.Sleep(200 * time.Millisecond)

		leader, n1, n2 := discoverLeader(t, node1, node2, node3)
		assert.True(t, leader.IsLeader())
		assert.False(t, n1.IsLeader())
		assert.False(t, n2.IsLeader())
		leader.Stop()
		time.Sleep(500 * time.Millisecond)
		assert.True(t, n1.IsLeader() || n2.IsLeader())
		assert.True(t, n1.monitor.IsMonitoring() || n2.monitor.IsMonitoring())
		n1.Stop()
		n2.Stop()
	})

	t.Run("A elected node should call the provider to create the resource", func(t *testing.T) {

		providerCalled := false
		provider := &mockProvider{
			fCreate: func(ctx context.Context) error {
				providerCalled = true
				return nil
			},
		}

		monitorHB := monitor.WithHeartBeat(igrpc.NewWatchClient(), 5*time.Millisecond)

		node1 := startNewNode(t, ":50051", &Config{
			Address: ":50051",
			Peers:   []string{":50052", ":50053"},
		}, provider, watcher.Config{
			MaxDelayForElection:    50,
			HeartBeatCheckInterval: 15 * time.Millisecond,
		}, monitorHB)

		node2 := startNewNode(t, ":50052", &Config{
			Address: ":50052",
			Peers:   []string{":50051", ":50053"},
		}, provider, defaultWc, monitorHB)

		node3 := startNewNode(t, ":50053", &Config{
			Address: ":50053",
			Peers:   []string{":50051", ":50052"},
		}, provider, defaultWc, monitorHB)

		time.Sleep(200 * time.Millisecond)

		leader, n1, n2 := discoverLeader(t, node1, node2, node3)

		assert.True(t, leader.IsLeader())
		assert.False(t, n1.IsLeader())
		assert.False(t, n2.IsLeader())
		leader.Stop()
		time.Sleep(500 * time.Millisecond)
		assert.True(t, n1.IsLeader() || n2.IsLeader())
		assert.True(t, providerCalled)
		n1.Stop()
		n2.Stop()
	})

	t.Run("The old leaader should call the provider to destroy the resources", func(t *testing.T) {

		providerCalled := false
		provider := &mockProvider{
			fCreate: func(ctx context.Context) error {
				return nil
			},
			fDestroy: func(ctx context.Context) error {
				providerCalled = true
				return nil
			},
		}

		monitorHB := monitor.WithHeartBeat(igrpc.NewWatchClient(), 5*time.Millisecond)

		node1 := startNewNode(t, ":50051", &Config{
			Address: ":50051",
			Peers:   []string{":50052", ":50053"},
		}, provider, watcher.Config{
			MaxDelayForElection:    50,
			HeartBeatCheckInterval: 15 * time.Millisecond,
		}, monitorHB)

		node2 := startNewNode(t, ":50052", &Config{
			Address: ":50052",
			Peers:   []string{":50051", ":50053"},
		}, provider, defaultWc, monitorHB)

		node3 := startNewNode(t, ":50053", &Config{
			Address: ":50053",
			Peers:   []string{":50051", ":50052"},
		}, provider, defaultWc, monitorHB)

		time.Sleep(200 * time.Millisecond)

		leader, n1, n2 := discoverLeader(t, node1, node2, node3)
		assert.True(t, leader.IsLeader())
		assert.False(t, n1.IsLeader())
		assert.False(t, n2.IsLeader())

		leader.Stop()
		time.Sleep(500 * time.Millisecond)
		assert.True(t, n1.IsLeader() || n2.IsLeader())
		go func() {
			err := leader.Start()
			if err != nil {
				t.Errorf("failed to start old leader %v", err)
			}
		}()
		time.Sleep(2 * time.Second)
		assert.True(t, providerCalled)

	})
}
