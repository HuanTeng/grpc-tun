package tun

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"

	pbtun "github.com/HuanTeng/grpc-tun/proto"
	log "github.com/sirupsen/logrus"
)

type tunnelClient struct {
	name   Addr
	cancel context.CancelFunc
	mu     sync.Mutex
	conn   chan net.Conn
}

type clientEndpoint struct {
	name   Addr
	stream pbtun.TunnelService_TunnelClient
	conn   *Conn

	recvChan chan []byte
	sendChan chan []byte
}

func NewClient(name string) (*tunnelClient, error) {
	return &tunnelClient{
		name: Addr(name),
		conn: make(chan net.Conn),
	}, nil
}

func (tc *tunnelClient) Serve(c pbtun.TunnelServiceClient) error {
	log.Debugf("new tunnel start")
	ctx, cancel := context.WithCancel(context.Background())
	tc.mu.Lock()
	if tc.cancel != nil {
		tc.cancel()
	}
	tc.cancel = cancel
	tc.mu.Unlock()

	stream, err := c.Tunnel(ctx)
	if err != nil {
		return err
	}
	if err := stream.Send(&pbtun.MessageToServer{
		MessageBody: &pbtun.MessageToServer_RegisterRequest{
			RegisterRequest: &pbtun.RegisterRequest{
				Name: string(tc.name),
			},
		},
	}); err != nil {
		return err
	}
	ep := &clientEndpoint{
		name:   tc.name,
		stream: stream,
		conn: &Conn{
			LocalNetAddr:  tc.name,
			RemoteNetAddr: Addr("_remote"),
		},
	}
	tc.conn <- ep.conn
	return ep.Serve()
}

func (tc *tunnelClient) Accept() (net.Conn, error) {
	conn, ok := <-tc.conn
	if !ok {
		return nil, errors.New("closed")
	}
	log.Debugf("Accept %s", conn.RemoteAddr())
	return conn, nil
}

func (tc *tunnelClient) Close() error {
	close(tc.conn)
	return nil
}

func (tc *tunnelClient) Addr() net.Addr {
	return tc.name
}

func (ep *clientEndpoint) Serve() error {
	stream := ep.stream
	ep.recvChan, ep.sendChan, _ = ep.conn.Open()

	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				// log.Debugf("received closed")
				close(ep.recvChan)
				return
			} else if err != nil {
				log.Warnf("received goroutine error: %v", err)
				close(ep.recvChan)
				return
			}
			payload := in.GetPayload()
			if payload != nil {
				// log.Debugf("received %d bytes payload", len(payload.Body))
				ep.recvChan <- payload.Body
			} else {
				log.Warnf("unexpected type %T", in.MessageBody)
			}
		}
	}()

	for data := range ep.sendChan {
		err := stream.Send(&pbtun.MessageToServer{
			MessageBody: &pbtun.MessageToServer_Payload{
				Payload: &pbtun.StreamPayload{
					Body: data,
				},
			},
		})
		if err != nil {
			ep.conn.Close()
		}
		// log.Debugf("sent %d bytes payload", len(data))
	}
	err := stream.CloseSend()
	// log.Debugf("tunnel finished")
	return err
}
