package netutil

import (
	"io"
	"net"

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

func ConnPIPE(c net.Conn, w *websocket.Conn) {
	pp := &connPIPE{conn: c, sock: w}
	pp.serve()
}

type connPIPE struct {
	conn net.Conn
	sock *websocket.Conn
}

func (pe *connPIPE) serve() {
	defer pe.close()

	go pe.connToSock()

	buf := make([]byte, 1024)
	for {
		_, rd, err := pe.sock.NextReader()
		if err != nil {
			_ = pe.conn.Close()
			break
		}
		n, err := rd.Read(buf)
		if err != nil {
			if err == io.EOF {
				continue
			}
			break
		}

		if _, err = pe.conn.Write(buf[:n]); err != nil {
			_ = pe.sock.Close()
		}
	}
}

func (pe *connPIPE) connToSock() {
	buf := make([]byte, 1024)
	for {
		n, err := pe.conn.Read(buf)
		if err != nil {
			_ = pe.sock.Close()
			break
		}
		if err = pe.sock.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
			_ = pe.conn.Close()
			break
		}
	}
}

func (pe *connPIPE) close() {
	_ = pe.conn.Close()
	_ = pe.sock.Close()
}
