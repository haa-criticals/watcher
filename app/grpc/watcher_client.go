package grpc

import (
	"context"
	"github.com.haa-criticals/watcher/app/grpc/pb"
	"github.com.haa-criticals/watcher/watcher"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
	"log"
	"time"
)

type Client struct {
	address string
}

func NewWatchClient(address string) *Client {
	return &Client{
		address: address,
	}
}

func (c *Client) RequestRegister(ctx context.Context, leaderAddress string, key string) (*watcher.RegisterResponse, error) {
	conn, err := grpc.Dial(leaderAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			log.Printf("failed to close connection: %v", err)
		}
	}(conn)

	client := pb.NewWatcherClient(conn)
	r, err := client.Register(ctx, &pb.RegisterRequest{Key: key, Address: c.address})
	if err != nil {
		return nil, err
	}

	nodes := make([]*watcher.NodeInfo, len(r.Nodes))
	for i, n := range r.Nodes {
		nodes[i] = &watcher.NodeInfo{Address: n.Address}
	}

	return &watcher.RegisterResponse{
		Success: r.Success,
		Nodes:   nodes,
	}, nil
}

func (c *Client) AckNode(ctx context.Context, address, key string, node *watcher.NodeInfo) (*watcher.NodeInfo, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			log.Printf("failed to close connection: %v", err)
		}
	}(conn)

	client := pb.NewWatcherClient(conn)
	res, err := client.AckNode(ctx, &pb.AckRequest{
		Key:  key,
		Node: &pb.Node{Address: node.Address},
	})

	if err != nil {
		return nil, err
	}

	return &watcher.NodeInfo{Address: res.Address}, err
}

func (c *Client) RequestElection(ctx context.Context, node, to, leader *watcher.NodeInfo, beat time.Time) (*watcher.ElectionResponse, error) {
	conn, err := grpc.Dial(to.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			log.Printf("failed to close connection: %v", err)
		}
	}(conn)

	client := pb.NewWatcherClient(conn)
	res, err := client.RequestElection(ctx, &pb.ElectionRequest{
		Requester: &pb.Node{
			Address: node.Address,
		},
		Leader: &pb.Node{
			Address: leader.Address,
		},
		LastBeat: timestamppb.New(beat),
	})
	if err != nil {
		return nil, err
	}

	return &watcher.ElectionResponse{
		Accepted: res.Accepted,
		Node:     &watcher.NodeInfo{Address: res.Node.Address},
	}, nil
}

func (c *Client) ElectionStart(ctx context.Context, node, to *watcher.NodeInfo, priority int32) error {
	conn, err := grpc.Dial(to.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			log.Printf("failed to close connection: %v", err)
		}
	}(conn)

	client := pb.NewWatcherClient(conn)
	_, err = client.ElectionStart(ctx, &pb.ElectionRegistration{
		Node: &pb.Node{
			Address: node.Address,
		},
		Priority: priority,
	})

	return err
}

func (c *Client) RequestElectionRegistration(ctx context.Context, node, to *watcher.NodeInfo, priority int32) error {
	conn, err := grpc.Dial(to.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			log.Printf("failed to close connection: %v", err)
		}
	}(conn)

	client := pb.NewWatcherClient(conn)
	_, err = client.RequestElectionRegistration(ctx, &pb.ElectionRegistration{
		Node: &pb.Node{
			Address: node.Address,
		},
		Priority: priority,
	})

	return err
}

func (c *Client) SendElectionVote(ctx context.Context, node, elected, to *watcher.NodeInfo) error {
	conn, err := grpc.Dial(to.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			log.Printf("failed to close connection: %v", err)
		}
	}(conn)

	client := pb.NewWatcherClient(conn)
	_, err = client.SendElectionVote(ctx, &pb.ElectionVote{
		Node: &pb.Node{
			Address: node.Address,
		},
		Elected: &pb.Node{
			Address: elected.Address,
		},
	})
	return err
}

func (c *Client) SendElectionConclusion(ctx context.Context, node, elected, to *watcher.NodeInfo) error {
	conn, err := grpc.Dial(to.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			log.Printf("failed to close connection: %v", err)
		}
	}(conn)

	client := pb.NewWatcherClient(conn)
	_, err = client.SendElectionConclusion(ctx, &pb.ElectedNode{
		Node: &pb.Node{
			Address: node.Address,
		},
		Elected: &pb.Node{
			Address: elected.Address,
		},
	})
	return err
}
