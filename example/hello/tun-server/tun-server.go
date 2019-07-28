package main

import (
	"context"
	"net"
	"time"

	tun "github.com/HuanTeng/grpc-tun"
	pbhello "github.com/HuanTeng/grpc-tun/example/hello/proto"
	pbtun "github.com/HuanTeng/grpc-tun/proto"
	log "github.com/sirupsen/logrus"

	"google.golang.org/grpc"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

func main() {
	tunnel, err := tun.NewServer()
	if err != nil {
		log.Fatalf("failed to create tunnel: %v", err)
	}

	go func() {
		lis, err := net.Listen("tcp", ":9000")
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		server := grpc.NewServer()
		pbtun.RegisterTunnelServiceServer(server, tunnel)
		server.Serve(lis)
	}()

	conn, err := grpc.Dial("name", grpc.WithInsecure(), tunnel.GetDialOption())
	if err != nil {
		log.Fatalf("failed to conn: %v", err)
	}
	defer conn.Close()
	client := pbhello.NewGreetingServiceClient(conn)
	for {
		time.Sleep(2 * time.Second)
		resp, err := client.Greeting(context.Background(), &pbhello.Person{
			Name: "Alice",
		})
		if err != nil {
			log.Printf("failed to conn: %v, will retry", err)
			continue
		}
		log.Printf("resp: %v", resp)
	}
}
