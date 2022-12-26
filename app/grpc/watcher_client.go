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

func (c *Client) RequestVote(ctx context.Context, address, candidate string, term int64) (*watcher.Vote, error) {
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
	res, err := client.RequestVote(ctx, &pb.Candidate{
		Requester: candidate,
		Term:      term,
	})

	if err != nil {
		return nil, err
	}

	return &watcher.Vote{
		Adress:  res.Node,
		Granted: res.Granted,
		Term:    res.Term,
	}, nil
}
