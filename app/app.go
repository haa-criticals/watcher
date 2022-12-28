package app

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	"net"

	"github.com.haa-criticals/watcher/app/grpc/pb"
	"github.com.haa-criticals/watcher/monitor"
	"github.com.haa-criticals/watcher/provisioner"
	"github.com.haa-criticals/watcher/watcher"
)

type Config struct {
	Address string
	Peers   []string
}

type App struct {
	pb.UnimplementedWatcherServer
	watcher     *watcher.Watcher
	monitor     *monitor.Monitor
	config      *Config
	provisioner *provisioner.Manager
	server      *grpc.Server
}

func New(w *watcher.Watcher, monitor *monitor.Monitor, provisioner *provisioner.Manager, config *Config) *App {
	w.Address = config.Address
	monitor.Address = config.Address
	w.RegisterNodes(config.Peers...)
	for _, peer := range config.Peers {
		monitor.RegisterWatcher(&watcher.Peer{Address: peer})
	}
	app := &App{
		watcher:     w,
		monitor:     monitor,
		provisioner: provisioner,
		config:      config,
	}

	w.OnElectionWon = func(w *watcher.Watcher, term int64) {
		monitor.NewTerm(term)
		go monitor.StartHeartBeating()
		go monitor.StartHealthChecks()
		err := provisioner.Create(context.Background())
		if err != nil {
			log.Println("failed to create resources", err)
		}
	}

	w.OnLostLeadership = func(w *watcher.Watcher, term int64) {
		log.Printf("%s Lost leadership in term %d", w.Address, term)
		go monitor.Stop()
		err := provisioner.Destroy(context.Background())
		if err != nil {
			log.Println("failed to destroy resources", err)
		}
	}

	return app
}

func (a *App) IsLeader() bool {
	return a.watcher.IsLeader()
}

func (a *App) Heartbeat(_ context.Context, beat *pb.Beat) (*emptypb.Empty, error) {
	a.watcher.OnReceiveHeartBeat(beat.Address, beat.Term, beat.Timestamp.AsTime())
	return &emptypb.Empty{}, nil
}

func (a *App) RequestVote(_ context.Context, request *pb.Candidate) (*pb.Vote, error) {
	vote := a.watcher.OnReceiveVoteRequest(&watcher.Candidate{
		Term:    request.Term,
		Address: request.Requester,
	})
	return &pb.Vote{
		Node:    a.watcher.Address,
		Term:    vote.Term,
		Granted: vote.Granted,
	}, nil
}

func (a *App) Start() error {
	go a.watcher.StartHeartBeatChecking()
	return a.StartServer()
}

func (a *App) StartServer() error {
	listen, err := net.Listen("tcp", a.config.Address)
	log.Println("Starting watcher server on ", listen.Addr())
	if err != nil {
		return err
	}
	a.server = grpc.NewServer()
	pb.RegisterWatcherServer(a.server, a)
	return a.server.Serve(listen)
}

func (a *App) Stop() {
	a.monitor.Stop()
	a.watcher.StopHeartBeatChecking()
	a.server.GracefulStop()
}
