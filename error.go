package tun

type ErrType struct {
	error
	IsTimeout   bool
	IsTemporary bool
}

func (e *ErrType) Timeout() bool {
	return e.IsTimeout
}

func (e *ErrType) Temporary() bool {
	return e.IsTemporary
}
