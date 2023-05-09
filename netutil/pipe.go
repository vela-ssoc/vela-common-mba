package netutil

import (
	"io"

	"github.com/gorilla/websocket"
)

func SocketPipe(a, b *websocket.Conn) {
	p := &socketPipe{a: a, b: b}
	p.running()
}

type socketPipe struct {
	a *websocket.Conn
	b *websocket.Conn
}

func (p *socketPipe) running() {
	defer func() {
		_ = p.a.Close()
		_ = p.b.Close()
	}()

	go p.pipeline(p.a, p.b)
	p.pipeline(p.b, p.a)
}

func (p *socketPipe) pipeline(a, b *websocket.Conn) {
	buf := make([]byte, 1024)
	for {
		mt, r, err := a.NextReader()
		if err != nil {
			break
		}
		w, err := b.NextWriter(mt)
		if err != nil {
			break
		}

		_, _ = io.CopyBuffer(w, r, buf)
		_ = w.Close()
	}
}
