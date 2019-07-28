package tun

type Addr string

func (a Addr) String() string {
	return string(a)
}

func (a Addr) Network() string {
	return "grpc-tun"
}
