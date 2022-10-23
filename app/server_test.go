package app

import (
	"context"
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

func TestRegisterWatcher(t *testing.T) {
	t.Run("should register watcher", func(t *testing.T) {
		config := &Config{
			Port: 50051,
		}
		server := New(watcher.New(igrpc.NewWatchClient("localhost:50051")), monitor.New(), config)
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
			config)
		go func() {
			err := leader.Start()
			if err != nil {
				t.Errorf("failed to start server: %v", err)
			}
		}()

		ref := time.Now()
		w := watcher.New(igrpc.NewWatchClient("localhost:50052"))
		node := New(w, monitor.New(), &Config{Port: 50052, Leader: "localhost:50051", Address: "localhost:50052"})
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
