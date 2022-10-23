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
		server := New(watcher.New(igrpc.NewWatchClient("localhost:50051")), monitor.New(), nil, config)
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
	})

	t.Run("Should receive heartbeat after registering", func(t *testing.T) {
		config := &Config{
			Port: 50051,
		}
		leader := New(
			watcher.New(igrpc.NewWatchClient("localhost:50051")),
			monitor.New(monitor.WithHeartBeat(igrpc.NewNotifier(), 1*time.Second)),
			nil,
			config)
		go func() {
			err := leader.Start()
			if err != nil {
				t.Errorf("failed to start server: %v", err)
			}
		}()

		ref := time.Now()
		w := watcher.New(igrpc.NewWatchClient("localhost:50052"))
		node := New(w, monitor.New(), nil, &Config{Port: 50052, Leader: "localhost:50051", Address: "localhost:50052"})
		go func() {
			err := node.Start()
			if err != nil {
				t.Errorf("failed to start node: %v", err)
			}
		}()
		go w.StartHeartBeatChecking()
		time.Sleep(3 * time.Second)
		assert.True(t, w.LastReceivedBeat().After(ref))
		w.StopHeartBeatChecking()
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
			watcher.New(igrpc.NewWatchClient("localhost:50051")),
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
			watcher.New(igrpc.NewWatchClient("localhost:50051")),
			monitor.New(monitor.WithHealthCheck(fmt.Sprintf("http://%s%s", server.Listener.Addr(), "/health"), 1*time.Second, 3)),
			provisioner.WithProvider(provider),
			config,
		)
		go func() {
			err := app.Start()
			if err != nil {
				t.Errorf("failed to start server: %v", err)
			}
		}()

		time.Sleep(2 * time.Second)
		assert.True(t, healthCalled)
		assert.True(t, app.monitor.IsHealthy())
		server.Close()
	})
}
