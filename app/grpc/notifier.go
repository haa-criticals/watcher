package grpc

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com.haa-criticals/watcher/app/grpc/pb"
)

type Notifier struct {
}

func (n *Notifier) Beat(watcherAddress string) error {
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

func NewNotifier() *Notifier {
	return &Notifier{}
}
