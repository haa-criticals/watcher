package grpc

import (
	"context"
	"github.com.haa-criticals/watcher/app/grpc/pb"
	"github.com.haa-criticals/watcher/watcher"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
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
		nodes[i] = &watcher.NodeInfo{Address: n.Address, ID: n.Id}
	}

	return &watcher.RegisterResponse{
		Success: r.Success,
		Id:      r.Id,
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
		Key: key,
		Node: &pb.Node{
			Id:      node.ID,
			Address: node.Address,
		},
	})

	if err != nil {
		return nil, err
	}

	return &watcher.NodeInfo{
		ID:      res.Id,
		Address: res.Address,
	}, err
}
