package grpc

import (
	"context"
	"google.golang.org/protobuf/types/known/timestamppb"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com.haa-criticals/watcher/app/grpc/pb"
	"github.com.haa-criticals/watcher/watcher"
)

type Client struct {
}

func NewWatchClient() *Client {
	return &Client{}
}

func (c *Client) Beat(watcherAddress string) error {
	conn, err := grpc.Dial(watcherAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
	_, err = client.Heartbeat(context.Background(), &pb.Beat{Timestamp: timestamppb.New(time.Now())})
	return err
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
