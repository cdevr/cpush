package options

import (
	"net"
	"time"
)

type Dialer func(network string, addr string) (net.Conn, error)

type Options struct {
	SuppressBanner  bool
	SuppressSending bool
	SuppressAdmin   bool
	SuppressOutput  bool
	Timeout         time.Duration
	Dialer          Dialer
}

func NewOptions() *Options {
	return &Options{
		Timeout: time.Second * 10,
	}
}

func (o *Options) Dial(network string, addr string) (net.Conn, error) {
	return o.Dialer(network, addr)
}
