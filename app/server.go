package app

import (
	"context"
	"github.com.haa-criticals/watcher/monitor"
	"github.com.haa-criticals/watcher/watcher"
	"net"

	"google.golang.org/grpc"

	"github.com.haa-criticals/watcher/app/grpc/pb"
)

type App struct {
	pb.UnimplementedWatcherServer
	monitor *monitor.Monitor
}

func New(monitor *monitor.Monitor) *App {
	return &App{
		monitor: monitor,
	}
}

func (a *App) Register(_ context.Context, in *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	a.monitor.RegisterWatcher(&watcher.Info{BaseURL: in.Address})
	return &pb.RegisterResponse{
		Success: true,
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
