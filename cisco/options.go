package cisco

import "time"

type Options struct {
	suppressBanner  bool
	suppressSending bool
	suppressAdmin   bool
	timeout         time.Duration
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

func (o *Options) SuppressBanner(v bool) *Options {
	o.suppressBanner = v
	return o
}

func (o *Options) Timeout(t time.Duration) *Options {
	o.timeout = t
	return o
}
