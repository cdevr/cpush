package options

import (
	"context"
	"net"
	"time"
)

type Dialer func(ctx context.Context, network string, addr string) (net.Conn, error)

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

func (o *Options) Dial(ctx context.Context, network string, addr string) (net.Conn, error) {
	return o.Dialer(ctx, network, addr)
}
