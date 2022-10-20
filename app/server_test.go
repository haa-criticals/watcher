package app

import (
	"context"
	"github.com.haa-criticals/watcher/monitor"
	"github.com.haa-criticals/watcher/watcher"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/stretchr/testify/assert"

	igrpc "github.com.haa-criticals/watcher/app/grpc"
	"github.com.haa-criticals/watcher/app/grpc/pb"
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

		r, err := client.Register(context.Background(), &pb.RegisterRequest{})
		assert.NoError(t, err)
		assert.NotNil(t, r)
		assert.True(t, r.Success)
	})
}
