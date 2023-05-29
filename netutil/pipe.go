package netutil

import (
	"io"
	"net"

	"github.com/gorilla/websocket"
)

func TwoSockPIPE(a, b *websocket.Conn) {
	p := &twoSock{a: a, b: b}
	p.exchange()
}

type twoSock struct {
	a, b *websocket.Conn
}

func (ts *twoSock) exchange() {
	defer ts.close()
	go ts.writeTo(ts.b, ts.a)
	ts.writeTo(ts.a, ts.b)
}

func (ts *twoSock) writeTo(dst, src *websocket.Conn) {
	buf := make([]byte, 1024)
	for {
		mt, rd, err := src.NextReader()
		if err != nil {
			_ = dst.Close()
			break
		}
		wt, err := dst.NextWriter(mt)
		if err != nil {
			_ = src.Close()
			break
		}
		if _, err = io.CopyBuffer(wt, rd, buf); err != nil {
			break
		}
	}
}

func (ts *twoSock) close() {
	_ = ts.a.Close()
	_ = ts.b.Close()
}

func ConnSockPIPE(c net.Conn, w *websocket.Conn) {
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
