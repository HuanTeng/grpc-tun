package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"

	pbhello "github.com/HuanTeng/grpc-tun/example/hello/proto"
)

type greetingServer struct{}

func (s *greetingServer) Greeting(ctx context.Context, person *pbhello.Person) (*pbhello.GreetingText, error) {
	return &pbhello.GreetingText{
		Text: fmt.Sprintf("Hello, %s.", person.Name),
	}, nil
}

func init() {
	flag.Parse()
}

func main() {
	lis, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	server := grpc.NewServer()
	pbhello.RegisterGreetingServiceServer(server, &greetingServer{})
	server.Serve(lis)
}
