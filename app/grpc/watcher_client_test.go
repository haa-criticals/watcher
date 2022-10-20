package grpc

import (
	"context"
	"github.com.haa-criticals/watcher/app/grpc/pb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"net"
	"testing"
	"time"
)

type MockWatcherServer struct {
	pb.UnimplementedWatcherServer
	fRegister func(ctx context.Context, in *pb.RegisterRequest) (*pb.RegisterResponse, error)
}

func (m *MockWatcherServer) Register(ctx context.Context, in *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	return m.fRegister(ctx, in)
}

func TestWatcherClient(t *testing.T) {
	t.Run("should register watcher", func(t *testing.T) {

		watcherServer := &MockWatcherServer{
			fRegister: func(ctx context.Context, in *pb.RegisterRequest) (*pb.RegisterResponse, error) {
				assert.Equal(t, "localhost:50051", in.Address)
				assert.Equal(t, "test", in.Key)
				return &pb.RegisterResponse{
					Success: true,
					Id:      "123",
				}, nil
			},
		}

		s := grpc.NewServer()
		pb.RegisterWatcherServer(s, watcherServer)
		go func() {
			listen, err := net.Listen("tcp", "localhost:50050")
			assert.NoError(t, err)
			err = s.Serve(listen)
			assert.NoError(t, err)
		}()

		time.Sleep(10 * time.Millisecond) // wait to start server
		c := NewWatchClient("localhost:50051")

		r, err := c.RequestRegister(context.Background(), "localhost:50050", "test")
		assert.NoError(t, err)
		assert.NotNil(t, r)
		assert.True(t, r.Success)
		assert.Equal(t, "123", r.Id)
	})

	t.Run("Should return all registered nodes", func(t *testing.T) {
		watcherServer := &MockWatcherServer{
			fRegister: func(ctx context.Context, in *pb.RegisterRequest) (*pb.RegisterResponse, error) {
				assert.Equal(t, "localhost:50051", in.Address)
				assert.Equal(t, "test", in.Key)
				return &pb.RegisterResponse{
					Success: true,
					Id:      "123",
					Nodes:   []string{"localhost:50051", "localhost:50052"},
				}, nil
			},
		}

		s := grpc.NewServer()
		pb.RegisterWatcherServer(s, watcherServer)
		go func() {
			listen, err := net.Listen("tcp", "localhost:50050")
			assert.NoError(t, err)
			err = s.Serve(listen)
			assert.NoError(t, err)
		}()

		time.Sleep(10 * time.Millisecond) // wait to start server
		c := NewWatchClient("localhost:50051")

		r, err := c.RequestRegister(context.Background(), "localhost:50050", "test")
		assert.NoError(t, err)
		assert.NotNil(t, r)
		assert.True(t, r.Success)
		assert.Equal(t, "123", r.Id)
		assert.Equal(t, []string{"localhost:50051", "localhost:50052"}, r.Nodes)
	})

	t.Run("Should return error when watcher is not available", func(t *testing.T) {
		c := NewWatchClient("localhost:50051")

		r, err := c.RequestRegister(context.Background(), "localhost:50050", "test")
		assert.Error(t, err)
		assert.Nil(t, r)
	})
}
