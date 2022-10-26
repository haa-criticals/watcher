package app

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	"net"

	"github.com.haa-criticals/watcher/app/grpc/pb"
	"github.com.haa-criticals/watcher/monitor"
	"github.com.haa-criticals/watcher/provisioner"
	"github.com.haa-criticals/watcher/watcher"
)

type Config struct {
	Port       int
	Leader     string
	clusterKey string
	Address    string
}

type App struct {
	pb.UnimplementedWatcherServer
	watcher     *watcher.Watcher
	monitor     *monitor.Monitor
	config      *Config
	isLeader    bool
	provisioner *provisioner.Manager
	server      *grpc.Server
}

func New(w *watcher.Watcher, monitor *monitor.Monitor, provisioner *provisioner.Manager, config *Config) *App {
	w.Address = config.Address
	return &App{
		watcher:     w,
		monitor:     monitor,
		provisioner: provisioner,
		config:      config,
		isLeader:    config.Leader == "",
	}
}

func (a *App) Register(_ context.Context, in *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	log.Printf("Registering node %s", in.Address)
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

func (a *App) Heartbeat(_ context.Context, in *pb.Beat) (*emptypb.Empty, error) {
	a.watcher.OnReceiveHeartBeat(in.Timestamp.AsTime())
	return &emptypb.Empty{}, nil
}

func (a *App) RequestElection(ctx context.Context, request *pb.ElectionRequest) (*pb.ElectionResponse, error) {
	requester := &watcher.NodeInfo{Address: request.Requester.Address}
	leader := &watcher.NodeInfo{Address: request.Leader.Address}

	res, err := a.watcher.OnNewElectionRequest(ctx, &watcher.ElectionRequest{
		Requester: requester,
		Leader:    leader,
		LastBeat:  request.LastBeat.AsTime(),
		StartedAt: request.StartedAt.AsTime(),
	})
	return &pb.ElectionResponse{
		Accepted: res.Accepted,
	}, err
}

func (a *App) ElectionStart(ctx context.Context, r *pb.ElectionRegistration) (*emptypb.Empty, error) {
	err := a.watcher.OnElectionStart(ctx, &watcher.NodeInfo{Address: r.Node.Address}, r.Priority)
	return &emptypb.Empty{}, err
}
func (a *App) RequestElectionRegistration(ctx context.Context, r *pb.ElectionRegistration) (*emptypb.Empty, error) {
	err := a.watcher.OnElectionRegistration(ctx, &watcher.NodeInfo{Address: r.Node.Address}, r.Priority)
	return &emptypb.Empty{}, err
}
func (a *App) SendElectionVote(ctx context.Context, vote *pb.ElectionVote) (*emptypb.Empty, error) {
	err := a.watcher.OnReceiveElectionVote(ctx,
		&watcher.NodeInfo{Address: vote.Node.Address},
		&watcher.NodeInfo{Address: vote.Elected.Address})
	return &emptypb.Empty{}, err
}
func (a *App) SendElectionConclusion(ctx context.Context, e *pb.ElectedNode) (*emptypb.Empty, error) {
	a.watcher.OnElectionConclusion(ctx,
		&watcher.NodeInfo{Address: e.Node.Address},
		&watcher.NodeInfo{Address: e.Elected.Address})
	return nil, status.Errorf(codes.Unimplemented, "method SendElectionConclusion not implemented")
}

func (a *App) Start() error {
	if !a.isLeader {
		go func() {
			err := a.requestRegister()
			if err != nil {
				log.Println("failed to register", err)
			}
		}()
	} else {
		log.Println("Starting watcher as leader on port")
		err := a.provisioner.Create(context.Background())
		if err != nil {
			return err
		}
		go a.monitor.StartHealthChecks()
	}
	return a.StartServer()
}

func (a *App) StartServer() error {
	listen, err := net.Listen("tcp", fmt.Sprintf(":%d", a.config.Port))
	log.Println("Starting watcher server on port", a.config.Port)
	if err != nil {
		return err
	}
	a.server = grpc.NewServer()
	pb.RegisterWatcherServer(a.server, a)
	return a.server.Serve(listen)
}

func (a *App) requestRegister() error {
	return a.watcher.RequestRegister(a.config.Leader, a.config.clusterKey)
}

func (a *App) Stop() {
	a.monitor.Stop()
	a.watcher.StopHeartBeatChecking()
	a.server.GracefulStop()
}
