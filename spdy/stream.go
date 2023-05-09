package spdy

import (
	"context"
	"io"
	"math"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type stream struct {
	id           uint32
	mux          *muxer
	syn          bool        // 是否已经发送了握手帧
	wmu          sync.Mutex  // 数据写锁
	rwn          sync.Mutex  // 数据读锁
	buff         []byte      // 消息缓冲池
	err          error       // 错误信息
	closed       atomic.Bool // 保证 close 方法只被执行一次
	ctx          context.Context
	cancel       context.CancelFunc
	readDeadline time.Time     // readDeadline
	readEvtCh    chan struct{} // 读取事件通知 channel
}

func (stm *stream) ID() uint32           { return stm.id }
func (stm *stream) LocalAddr() net.Addr  { return stm.mux.LocalAddr() }
func (stm *stream) RemoteAddr() net.Addr { return stm.mux.RemoteAddr() }

func (stm *stream) SetDeadline(t time.Time) (err error) {
	if err = stm.SetReadDeadline(t); err == nil {
		err = stm.SetWriteDeadline(t)
	}
	return
}

func (stm *stream) SetReadDeadline(t time.Time) error {
	stm.readDeadline = t
	stm.notifyReadEvt()
	return nil
}

func (stm *stream) SetWriteDeadline(time.Time) error { return nil }

func (stm *stream) Close() error {
	return stm.closeError(io.EOF, true)
}

func (stm *stream) receive(p []byte) (int, error) {
	total := len(p)
	if total == 0 {
		return 0, nil
	}

	stm.rwn.Lock()
	stm.buff = append(stm.buff, p...)
	stm.rwn.Unlock()

	return total, nil
}

func (stm *stream) Write(p []byte) (int, error) {
	psz := len(p)
	if psz == 0 {
		return 0, nil
	}

	const maximum = math.MaxUint16 // 每帧最大传输 65535 个字节

	// 检查 stream 是否已经关闭
	select {
	case <-stm.ctx.Done():
		return 0, io.ErrClosedPipe
	default:
	}

	stm.wmu.Lock()
	defer stm.wmu.Unlock()

	flag := flagDAT
	if !stm.syn {
		stm.syn = true
		flag = flagSYN
	}

	n := psz
	for n > 0 {
		if n > maximum {
			n = maximum
		}

		if _, err := stm.mux.write(flag, stm.id, p[:n]); err != nil {
			return 0, err
		}

		flag = flagDAT
		p = p[n:]
		n = len(p)
	}

	return psz, nil
}

func (stm *stream) Read(p []byte) (int, error) {
	for {
		if block, n := stm.read(p); !block {
			return n, nil
		}
		if err := stm.readBlocking(); err != nil {
			return 0, err
		}
	}
}

func (stm *stream) read(p []byte) (bool, int) {
	if i := len(p); i == 0 {
		return false, 0
	}
	stm.rwn.Lock()
	defer stm.rwn.Unlock()

	if len(stm.buff) > 0 {
		n := copy(p, stm.buff)
		stm.buff = stm.buff[n:]
		return false, n
	}

	return true, 0
}

func (stm *stream) readBlocking() error {
	var deadline <-chan time.Time
	if dead := stm.readDeadline; !dead.IsZero() {
		timer := time.NewTimer(time.Until(dead))
		defer timer.Stop()
		deadline = timer.C
	}

	select {
	case <-stm.readEvtCh:
		return nil
	case <-deadline:
		return context.DeadlineExceeded
	case <-stm.ctx.Done():
		return stm.ctx.Err()
	}
}

func (stm *stream) closeError(err error, fin bool) error {
	if !stm.closed.CompareAndSwap(false, true) {
		return io.ErrClosedPipe
	}

	stmID := stm.id
	stm.mux.delStream(stmID)

	if fin && stm.syn {
		_, _ = stm.mux.write(flagFIN, stmID, nil)
	}

	stm.err = err
	stm.cancel()

	return err
}

func (stm *stream) notifyReadEvt() {
	select {
	case stm.readEvtCh <- struct{}{}:
	default:
	}
}
