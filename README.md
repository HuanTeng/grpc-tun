# grpc-tun
gRPC reverse tunnel over gRPC streaming

## Usage

See `example/hello`.

`example/hello/proto` defines an example gRPC service `GreetingService`.

`example/hello/tun-server` is a tunnel server and `GreetingService` client.

`example/hello/tun-client` is a tunnel client and `GreetingService` server.

