package spdy

import (
	"context"
	"net"
	"time"
)

type option struct {
	maxsize  int
	backlog  int
	interval time.Duration
	capacity int
	server   bool
	passwd   []byte
}

type Option func(*option)

func WithReadTimout(du time.Duration) Option {
	return func(opt *option) {
		opt.interval = du
	}
}

func WithBacklog(n int) Option {
	return func(opt *option) {
		opt.backlog = n
	}
}

func WithMaxsize(n int) Option {
	return func(opt *option) {
		opt.maxsize = n
	}
}

func WithCapacity(n int) Option {
	return func(opt *option) {
		opt.capacity = n
	}
}

func WithEncrypt(passwd []byte) Option {
	return func(opt *option) {
		opt.passwd = passwd
	}
}

func (opt option) muxer(tran net.Conn) *muxer {
	backlog := opt.backlog
	capacity := opt.capacity
	if backlog < 0 {
		backlog = 0
	}
	if capacity <= 0 {
		capacity = 64
	}

	ctx, cancel := context.WithCancel(context.Background())
	mux := &muxer{
		tran:     tran,
		interval: opt.interval,
		streams:  make(map[uint32]*stream, capacity),
		accepts:  make(chan *stream, backlog),
		passwd:   opt.passwd,
		ctx:      ctx,
		cancel:   cancel,
	}
	if opt.server {
		mux.stmID.Add(1)
	}

	return mux
}
