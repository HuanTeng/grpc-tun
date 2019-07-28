package tun

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

// Conn implements net.Conn
type Conn struct {
	LocalNetAddr  net.Addr
	RemoteNetAddr net.Addr

	mu     sync.Mutex
	isOpen bool

	recvBuf    []byte
	recvChan   chan []byte
	recvCtx    context.Context
	recvMu     sync.Mutex
	cancelRecv context.CancelFunc

	sendChan   chan []byte
	sendCtx    context.Context
	cancelSend context.CancelFunc
}

func (c *Conn) Open() (recv, send chan []byte, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.isOpen {
		return nil, nil, &ErrType{
			error: errors.New("already open"),
		}
	}
	c.isOpen = true
	c.recvChan, c.sendChan = make(chan []byte), make(chan []byte)
	c.recvCtx, c.cancelRecv = context.WithCancel(context.Background())
	c.sendCtx, c.cancelSend = context.WithCancel(context.Background())
	return c.recvChan, c.sendChan, nil
}

func (c *Conn) Read(b []byte) (n int, err error) {
	if !c.isOpen {
		err = &ErrType{
			error: errors.New("conn not open"),
		}
		return
	}
	c.recvMu.Lock()
	defer c.recvMu.Unlock()
	n = copy(b, c.recvBuf)
	b = b[n:]
	if len(b) == 0 {
		// log.Debugf("Read %d bytes: %s", n, hex.EncodeToString(b0[:n]))
		return
	}
	c.recvBuf = nil
	select {
	case <-c.recvCtx.Done():
		err1 := &ErrType{
			error: c.recvCtx.Err(),
		}
		if c.recvCtx.Err() == context.DeadlineExceeded {
			err1.IsTimeout = true
		}
		err = err1
	case recv, ok := <-c.recvChan:
		if !ok {
			err = io.EOF
		} else {
			n1 := copy(b, recv)
			c.recvBuf = recv[n1:]
			n += n1
		}
	}
	// log.Debugf("Read %d bytes: %s", n, hex.EncodeToString(b0[:n]))
	return
}

func (c *Conn) Write(b []byte) (n int, err error) {
	// log.Debugf("Beging Write %d bytes: %s", len(b), hex.EncodeToString(b))
	if !c.isOpen {
		err = &ErrType{
			error: errors.New("conn not open"),
		}
		return
	}
	bCopy := make([]byte, len(b))
	copy(bCopy, b)
	select {
	case <-c.sendCtx.Done():
		err1 := &ErrType{
			error: c.sendCtx.Err(),
		}
		if c.sendCtx.Err() == context.DeadlineExceeded {
			err1.IsTimeout = true
		}
		err = err1
	case c.sendChan <- bCopy:
		n = len(b)
	}
	// log.Debugf("Wrote %d bytes", n)
	return
}

func (c *Conn) Close() error {
	// log.Debugf("Close")
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.isOpen {
		return &ErrType{
			error: errors.New("already closed"),
		}
	}
	c.isOpen = false

	c.cancelRecv()
	c.cancelSend()
	close(c.sendChan)
	return nil
}

func (c *Conn) LocalAddr() net.Addr {
	return c.LocalNetAddr
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.RemoteNetAddr
}

func (c *Conn) SetDeadline(t time.Time) error {
	c.SetReadDeadline(t)
	c.SetWriteDeadline(t)
	return nil
}

func (c *Conn) SetReadDeadline(t time.Time) error {
	if t.IsZero() {
		c.recvCtx, c.cancelRecv = context.WithCancel(context.Background())
	} else {
		c.recvCtx, c.cancelRecv = context.WithDeadline(c.recvCtx, t)
	}
	return nil
}

func (c *Conn) SetWriteDeadline(t time.Time) error {
	if t.IsZero() {
		c.sendCtx, c.cancelSend = context.WithCancel(context.Background())
	} else {
		c.sendCtx, c.cancelSend = context.WithDeadline(c.recvCtx, t)
	}
	return nil
}
