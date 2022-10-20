package app

import (
	"context"
	"fmt"
	"github.com.haa-criticals/watcher/monitor"
	"github.com.haa-criticals/watcher/watcher"
	"log"
	"net"

	"google.golang.org/grpc"

	"github.com.haa-criticals/watcher/app/grpc/pb"
)

type Config struct {
	Port           int
	Leader         bool
	LeaderEndpoint string
	clusterKey     string
}

type App struct {
	pb.UnimplementedWatcherServer
	watcher *watcher.Watcher
	monitor *monitor.Monitor
	config  *Config
}

func New(watcher *watcher.Watcher, monitor *monitor.Monitor, config *Config) *App {
	return &App{
		watcher: watcher,
		monitor: monitor,
		config:  config,
	}
}

func (a *App) Register(_ context.Context, in *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	log.Printf("Registering node  %s", in.Address)
	nodeInfo := &watcher.NodeInfo{Address: in.Address}
	registerdNodes, err := a.watcher.RegisterNode(nodeInfo, in.Key)
	if err != nil {
		return &pb.RegisterResponse{
			Success: false,
		}, err
	}
	a.monitor.RegisterWatcher(nodeInfo)

	nodes := make([]*pb.Node, len(registerdNodes))
	for i, n := range registerdNodes {
		nodes[i] = &pb.Node{
			Id:      n.ID,
			Address: n.Address,
		}
	}

	log.Printf("Registered node %s", in.Address)

	return &pb.RegisterResponse{
		Success: true,
		Id:      nodeInfo.ID,
		Nodes:   nodes,
	}, nil
}

func (a *App) Start() error {
	if !a.config.Leader {
		go a.requestRegister()
	}
	return a.StartServer()
}

func (a *App) StartServer() error {
	listen, err := net.Listen("tcp", fmt.Sprintf(":%d", a.config.Port))
	log.Println("Starting watcher server on port", a.config.Port)
	if err != nil {
		return err
	}
	s := grpc.NewServer()
	pb.RegisterWatcherServer(s, a)
	return s.Serve(listen)
}

func (a *App) requestRegister() {
	err := a.watcher.RequestRegister(a.config.LeaderEndpoint, a.config.clusterKey)
	if err != nil {
		log.Println("Error registering node", err)
	}
}
