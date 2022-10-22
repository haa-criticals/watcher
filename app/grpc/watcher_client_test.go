package grpc

import (
	"context"
	"github.com.haa-criticals/watcher/app/grpc/pb"
	"github.com.haa-criticals/watcher/watcher"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"net"
	"testing"
	"time"
)

type mockWatcherServer struct {
	pb.UnimplementedWatcherServer
	fRegister func(ctx context.Context, in *pb.RegisterRequest) (*pb.RegisterResponse, error)
	fAckNode  func(ctx context.Context, in *pb.AckRequest) (*pb.Node, error)
}

func (m *mockWatcherServer) Register(ctx context.Context, in *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	return m.fRegister(ctx, in)
}

func (m *mockWatcherServer) AckNode(ctx context.Context, in *pb.AckRequest) (*pb.Node, error) {
	return m.fAckNode(ctx, in)
}

func TestRequestRegisterWatcherClient(t *testing.T) {
	t.Run("should register watcher", func(t *testing.T) {

		watcherServer := &mockWatcherServer{
			fRegister: func(ctx context.Context, in *pb.RegisterRequest) (*pb.RegisterResponse, error) {
				assert.Equal(t, "localhost:50051", in.Address)
				assert.Equal(t, "test", in.Key)
				return &pb.RegisterResponse{
					Success: true,
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
		s.GracefulStop()
	})

	t.Run("Should return all registered nodes", func(t *testing.T) {
		watcherServer := &mockWatcherServer{
			fRegister: func(ctx context.Context, in *pb.RegisterRequest) (*pb.RegisterResponse, error) {
				assert.Equal(t, "localhost:50051", in.Address)
				assert.Equal(t, "test", in.Key)
				return &pb.RegisterResponse{
					Success: true,
					Nodes: []*pb.Node{
						{Address: "localhost:50051"},
						{Address: "localhost:50052"},
					},
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
		assert.Equal(t, 2, len(r.Nodes))
		assert.Equal(t, "localhost:50051", r.Nodes[0].Address)
		assert.Equal(t, "localhost:50052", r.Nodes[1].Address)
		s.GracefulStop()
	})

	t.Run("Should return error when watcher is not available", func(t *testing.T) {
		c := NewWatchClient("localhost:50051")

		r, err := c.RequestRegister(context.Background(), "localhost:50050", "test")
		assert.Error(t, err)
		assert.Nil(t, r)
	})
}

func TestAckWatcherClient(t *testing.T) {
	t.Run("Should return error when acked node is not available", func(t *testing.T) {
		c := NewWatchClient("localhost:50051")

		r, err := c.AckNode(context.Background(), "localhost:50050", "test", &watcher.NodeInfo{Address: "localhost:50051"})
		assert.Error(t, err)
		assert.Nil(t, r)
	})

	t.Run("Should return the acked node info", func(t *testing.T) {
		watcherServer := &mockWatcherServer{
			fAckNode: func(ctx context.Context, in *pb.AckRequest) (*pb.Node, error) {
				assert.Equal(t, "localhost:50051", in.Node.Address)
				assert.Equal(t, "test", in.Key)
				return &pb.Node{
					Address: "localhost:50050",
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

		r, err := c.AckNode(context.Background(), "localhost:50050", "test", &watcher.NodeInfo{Address: "localhost:50051"})
		assert.NoError(t, err)
		assert.NotNil(t, r)
		assert.Equal(t, "localhost:50050", r.Address)
		s.GracefulStop()
	})
}
