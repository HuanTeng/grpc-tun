package main

import (
	"context"
	"log"
	"time"

	pbhello "github.com/HuanTeng/grpc-tun/example/hello/proto"

	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:9000", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to conn: %v", err)
	}
	defer conn.Close()
	client := pbhello.NewGreetingServiceClient(conn)
	for {
		resp, err := client.Greeting(context.Background(), &pbhello.Person{
			Name: "Alice",
		})
		if err != nil {
			log.Fatalf("failed to conn: %v", err)
		}
		log.Printf("resp: %v", resp)
		time.Sleep(2 * time.Second)
	}
}
