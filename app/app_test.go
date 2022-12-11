package app

import (
	"context"
	"fmt"
	"github.com.haa-criticals/watcher/provisioner"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/stretchr/testify/assert"

	igrpc "github.com.haa-criticals/watcher/app/grpc"
	"github.com.haa-criticals/watcher/app/grpc/pb"
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

func TestRegisterWatcher(t *testing.T) {
	t.Run("should register watcher", func(t *testing.T) {
		config := &Config{
			Port: 50051,
		}

		p := provisioner.WithProvider(&mockProvider{
			fCreate: func(ctx context.Context) error {
				return nil
			},
		})

		w := watcher.New(igrpc.NewWatchClient("localhost:50051"), watcher.Config{})

		server := New(w, monitor.New(), p, config)
		go func() {
			err := server.Start()
			if err != nil {
				t.Errorf("failed to start server: %v", err)
			}
		}()

		conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
		assert.NoError(t, err)
		client := pb.NewWatcherClient(conn)

		r, err := client.Register(context.Background(), &pb.RegisterRequest{Address: "localhost:50052"})
		assert.NoError(t, err)
		assert.NotNil(t, r)
		assert.True(t, r.Success)
		server.Stop()
	})

	t.Run("Should receive heartbeat after registering", func(t *testing.T) {
		config := &Config{
			Port: 50051,
		}
		p := provisioner.WithProvider(&mockProvider{fCreate: func(ctx context.Context) error {
			return nil
		}})

		leader := New(
			watcher.New(igrpc.NewWatchClient("localhost:50051"), watcher.Config{}),
			monitor.New(monitor.WithHeartBeat(igrpc.NewNotifier(), 3*time.Millisecond)),
			p,
			config)
		go func() {
			err := leader.Start()
			if err != nil {
				t.Errorf("failed to start server: %v", err)
			}
		}()

		ref := time.Now()
		w := watcher.New(igrpc.NewWatchClient("localhost:50052"), watcher.Config{})
		node := New(w, monitor.New(), nil, &Config{Port: 50052, Leader: "localhost:50051", Address: "localhost:50052"})
		go func() {
			err := node.Start()
			if err != nil {
				t.Errorf("failed to start node: %v", err)
			}
		}()
		go w.StartHeartBeatChecking()
		time.Sleep(9 * time.Millisecond)
		assert.True(t, w.LastReceivedBeat().After(ref))
		w.StopHeartBeatChecking()
		leader.Stop()
		node.Stop()
	})
}

func TestAppStart(t *testing.T) {
	t.Run("Should call the provisioner create when the app starts as leader", func(t *testing.T) {
		config := &Config{
			Port: 50051,
		}

		createCalled := false
		provider := &mockProvider{
			fCreate: func(ctx context.Context) error {
				createCalled = true
				return nil
			},
		}

		server := New(
			watcher.New(igrpc.NewWatchClient("localhost:50051"), watcher.Config{}),
			monitor.New(),
			provisioner.WithProvider(provider),
			config,
		)
		go func() {
			err := server.Start()
			if err != nil {
				t.Errorf("failed to start server: %v", err)
			}
		}()

		time.Sleep(2 * time.Second)
		assert.True(t, createCalled)
		server.Stop()

	})

	t.Run("Should start health check when the app starts as leader", func(t *testing.T) {
		config := &Config{
			Port: 50051,
		}

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
				log.Println("server started")
				return nil
			},
		}

		app := New(
			watcher.New(igrpc.NewWatchClient("localhost:50051"), watcher.Config{}),
			monitor.New(monitor.WithHealthCheck(fmt.Sprintf("http://%s%s", server.Listener.Addr(), "/health"), 5*time.Millisecond, 3)),
			provisioner.WithProvider(provider),
			config,
		)
		go func() {
			err := app.Start()
			if err != nil {
				t.Errorf("failed to start server: %v", err)
			}
		}()

		time.Sleep(10 * time.Millisecond)
		assert.True(t, healthCalled)
		assert.True(t, app.monitor.IsHealthy())
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

		leader := New(
			watcher.New(igrpc.NewWatchClient("localhost:50051"), watcher.Config{}),
			monitor.New(monitor.WithHeartBeat(igrpc.NewNotifier(), 5*time.Millisecond)),
			provisioner.WithProvider(provider),
			&Config{Port: 50051, Address: "localhost:50051"},
		)
		go func() {
			err := leader.Start()
			if err != nil {
				t.Errorf("failed to start server: %v", err)
			}
		}()

		node1 := New(
			watcher.New(igrpc.NewWatchClient("localhost:50052"), watcher.Config{}),
			monitor.New(),
			provisioner.WithProvider(provider),
			&Config{Port: 50052, Leader: "localhost:50051", Address: "localhost:50052"},
		)

		go func() {
			err := node1.Start()
			if err != nil {
				t.Errorf("failed to start node: %v", err)
			}
		}()

		node2 := New(
			watcher.New(igrpc.NewWatchClient("localhost:50053"), watcher.Config{}),
			monitor.New(),
			provisioner.WithProvider(provider),
			&Config{Port: 50053, Leader: "localhost:50051", Address: "localhost:50053"},
		)

		go func() {
			err := node2.Start()
			if err != nil {
				t.Errorf("failed to start node: %v", err)
			}
		}()

		time.Sleep(10 * time.Millisecond)
		assert.True(t, leader.isLeader)
		assert.False(t, node1.isLeader)
		assert.False(t, node2.isLeader)
		leader.Stop()
		time.Sleep(15 * time.Millisecond)
		assert.True(t, node1.isLeader || node2.isLeader)
		node1.Stop()
		node2.Stop()
	})
}
