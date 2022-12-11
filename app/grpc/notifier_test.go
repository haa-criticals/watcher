package grpc

import (
	"context"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com.haa-criticals/watcher/app/grpc/pb"
)

type recorderServer struct {
	pb.UnimplementedWatcherServer
	beats  []time.Time
	server *grpc.Server
}

func (r *recorderServer) Heartbeat(_ context.Context, beat *pb.Beat) (*emptypb.Empty, error) {
	r.beats = append(r.beats, beat.Timestamp.AsTime())
	return &emptypb.Empty{}, nil
}

func startRecorderServer(server *recorderServer) error {
	listen, err := net.Listen("tcp", ":50051")
	if err != nil {
		return err
	}
	s := grpc.NewServer()
	pb.RegisterWatcherServer(s, server)
	server.server = s
	return s.Serve(listen)
}

func TestNotifier(t *testing.T) {
	t.Run("should notify watcher", func(t *testing.T) {
		server := &recorderServer{}
		go func() {
			err := startRecorderServer(server)
			if err != nil {
				t.Errorf("failed to start server: %v", err)
			}
		}()

		notifier := NewNotifier()
		err := notifier.Beat("localhost:50051")
		assert.NoError(t, err)
		assert.Len(t, server.beats, 1)
		server.server.Stop()
	})
}
