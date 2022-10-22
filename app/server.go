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
	Port       int
	Leader     string
	clusterKey string
	Address    string
}

type App struct {
	pb.UnimplementedWatcherServer
	watcher *watcher.Watcher
	monitor *monitor.Monitor
	config  *Config
}

func New(watcher *watcher.Watcher, monitor *monitor.Monitor, config *Config) *App {
	watcher.Address = config.Address
	return &App{
		watcher: watcher,
		monitor: monitor,
		config:  config,
	}
}

func (a *App) Register(_ context.Context, in *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	log.Printf("Registering node  %s", in.Address)
	nodeInfo := &watcher.NodeInfo{Address: in.Address}
	registeredNodes, err := a.watcher.RegisterNode(nodeInfo, in.Key)
	if err != nil {
		return &pb.RegisterResponse{
			Success: false,
		}, err
	}
	a.monitor.RegisterWatcher(nodeInfo)

	nodes := make([]*pb.Node, len(registeredNodes))
	for i, n := range registeredNodes {
		nodes[i] = &pb.Node{Address: n.Address}
	}

	log.Printf("Registered node %s", in.Address)
	log.Println("Registered nodes", nodes)
	return &pb.RegisterResponse{
		Success: true,
		Nodes:   nodes,
	}, nil
}

func (a *App) AckNode(_ context.Context, in *pb.AckRequest) (*pb.Node, error) {
	log.Printf("Acknowledging node %s", in.Node.Address)
	nodeInfo := &watcher.NodeInfo{Address: in.Node.Address}
	err := a.watcher.AckNode(nodeInfo, in.Key)
	if err != nil {
		return nil, err
	}
	log.Printf("Acknowledged node %s", in.Node.Address)
	return &pb.Node{
		Address: a.watcher.Address,
	}, nil
}

func (a *App) Start() error {
	if a.config.Leader != "" {
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
	err := a.watcher.RequestRegister(a.config.Leader, a.config.clusterKey)
	if err != nil {
		log.Println("Error registering node", err)
	}
}
