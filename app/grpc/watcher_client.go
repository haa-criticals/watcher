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

	return &watcher.RegisterResponse{
		Success: r.Success,
		Id:      r.Id,
		Nodes:   r.Nodes,
	}, nil
}
