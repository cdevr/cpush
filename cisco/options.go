package cisco

import (
	"net"
	"time"
)

type Dialer func(network string, addr string) (net.Conn, error)

type Options struct {
	suppressBanner  bool
	suppressSending bool
	suppressAdmin   bool
	suppressOutput  bool
	timeout         time.Duration
	dialer          Dialer
}

func NewOptions() *Options {
	return &Options{
		timeout: time.Second * 10,
	}
}

func (o *Options) SuppressSending(v bool) *Options {
	o.suppressSending = v
	return o
}

func (o *Options) SuppressAdmin(v bool) *Options {
	o.suppressAdmin = v
	return o
}

func (o *Options) SuppressOutput(v bool) *Options {
	o.suppressOutput = v
	return o
}

func (o *Options) SuppressBanner(v bool) *Options {
	o.suppressBanner = v
	return o
}

func (o *Options) Timeout(t time.Duration) *Options {
	o.timeout = t
	return o
}

func (o *Options) Dialer(d Dialer) *Options {
	o.dialer = d
	return o
}
