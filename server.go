package tun

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	pbtun "github.com/HuanTeng/grpc-tun/proto"
)

type serverEndpoint struct {
	name string
	conn *Conn

	recvChan chan []byte
	sendChan chan []byte
}

type tunnelServer struct {
	eps      map[string]*serverEndpoint
	epsMutex sync.RWMutex
}

func NewServer() (*tunnelServer, error) {
	return &tunnelServer{
		eps: make(map[string]*serverEndpoint),
	}, nil
}

func (ts *tunnelServer) Tunnel(stream pbtun.TunnelService_TunnelServer) error {
	log.Debugf("new tunnel start")
	reg, err := stream.Recv()
	if err == io.EOF {
		log.Printf("server: expect register request, got EOF")
		return nil
	}
	regReq := reg.GetRegisterRequest()
	if regReq == nil {
		log.Printf("server: expect register request, got nil")
		return nil
	}
	log.Debugf("new server endpoint name: %s", regReq.Name)
	ep := &serverEndpoint{
		name: regReq.Name,
		conn: &Conn{
			LocalNetAddr:  Addr("_local"),
			RemoteNetAddr: Addr(regReq.Name),
		},
	}
	ep.recvChan, ep.sendChan, err = ep.conn.Open()

	ts.epsMutex.Lock()
	ts.eps[ep.name] = ep
	ts.epsMutex.Unlock()

	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				// log.Debugf("received goroutine closed")
				close(ep.recvChan)
				return
			} else if err != nil {
				log.Warnf("received goroutine error: %v", err)
				close(ep.recvChan)
				return
			}
			payload := in.GetPayload()
			if payload != nil {
				// log.Debugf("received %d bytes payload: %s", len(payload.Body), hex.EncodeToString(payload.Body))
				ep.recvChan <- payload.Body
			} else {
				log.Warnf("unexpected type %T", in.MessageBody)
			}
		}
	}()

	for data := range ep.sendChan {
		err := stream.Send(&pbtun.MessageFromServer{
			MessageBody: &pbtun.MessageFromServer_Payload{
				Payload: &pbtun.StreamPayload{
					Body: data,
				},
			},
		})
		if err != nil {
			ep.conn.Close()
		}
		// log.Debugf("sent %d bytes payload: %s", len(data), hex.EncodeToString(data))
	}
	// log.Debugf("tunnel finished")

	return nil
}

func (ts *tunnelServer) Dial(network, address string) (net.Conn, error) {
	// log.Debugf("Dial network=%s %s", network, address)
	if network != "grpc-tun" {
		err1 := &ErrType{
			error: errors.New("unknown network"),
		}
		return nil, &net.OpError{Op: "dial", Net: network, Source: nil, Addr: nil, Err: err1}
	}

	ts.epsMutex.RLock()
	defer ts.epsMutex.RUnlock()

	if c, ok := ts.eps[address]; ok {
		return c.conn, nil
	}
	err1 := &ErrType{
		error:       errors.New("unknown name"),
		IsTemporary: true,
	}
	return nil, &net.OpError{Op: "dial", Net: network, Source: nil, Addr: Addr(address), Err: err1}
}

func (ts *tunnelServer) GetDialOption() grpc.DialOption {
	return grpc.WithContextDialer(func(ctx context.Context, name string) (net.Conn, error) {
		return ts.Dial("grpc-tun", name)
	})
}
