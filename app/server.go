package app

import (
	"context"
	"github.com.haa-criticals/watcher/watcher"
	"net"

	"google.golang.org/grpc"

	"github.com.haa-criticals/watcher/app/grpc/pb"
)

type App struct {
	pb.UnimplementedWatcherServer
	watcher *watcher.Watcher
}

func New(watcher *watcher.Watcher) *App {
	return &App{
		watcher: watcher,
	}
}

func (a *App) Register(_ context.Context, in *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	nodeInfo := &watcher.NodeInfo{BaseURL: in.Address}
	nodes, err := a.watcher.RegisterNode(nodeInfo, in.Key)
	if err != nil {
		return &pb.RegisterResponse{
			Success: false,
		}, err
	}

	nodesAddress := make([]string, len(nodes))
	for i, n := range nodes {
		nodesAddress[i] = n.BaseURL
	}

	return &pb.RegisterResponse{
		Success: true,
		Id:      nodeInfo.ID.String(),
		Nodes:   nodesAddress,
	}, nil
}

func (a *App) StartServer() error {
	listen, err := net.Listen("tcp", ":50051")
	if err != nil {
		return err
	}
	s := grpc.NewServer()
	pb.RegisterWatcherServer(s, a)
	return s.Serve(listen)
}
