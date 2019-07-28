package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	tun "github.com/HuanTeng/grpc-tun"
	pbhello "github.com/HuanTeng/grpc-tun/example/hello/proto"
	pbtun "github.com/HuanTeng/grpc-tun/proto"
	log "github.com/sirupsen/logrus"

	"google.golang.org/grpc"
)

type greetingServer struct{}

func (s *greetingServer) Greeting(ctx context.Context, person *pbhello.Person) (*pbhello.GreetingText, error) {
	log.Printf("Got Greeting request: %s", person.Name)
	return &pbhello.GreetingText{
		Text: fmt.Sprintf("Hello, %s.", person.Name),
	}, nil
}

func init() {
	flag.Parse()
	log.SetLevel(log.DebugLevel)
}

func main() {
	tunnel, err := tun.NewClient("name")
	if err != nil {
		log.Fatalf("failed to create tunnel: %v", err)
	}

	go func() {
		conn, err := grpc.Dial("localhost:9000", grpc.WithInsecure())
		if err != nil {
			log.Fatalf("failed to conn: %v", err)
		}
		defer conn.Close()
		client := pbtun.NewTunnelServiceClient(conn)
		for {
			err := tunnel.Serve(client)
			if err == nil {
				log.Printf("tunnel closed normally")
				break
			}
			log.Printf("tunnel serve error: %v", err)
			time.Sleep(1 * time.Second)
		}
	}()

	server := grpc.NewServer()
	pbhello.RegisterGreetingServiceServer(server, &greetingServer{})
	server.Serve(tunnel)
}
