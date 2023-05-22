package taskpool

import (
	"runtime"
	"sync"
)

type RunnerFunc func()

func (f RunnerFunc) Run() { f() }

type Runner interface {
	Run()
}

type Executor interface {
	Submit(r Runner)
}

// NewPool 创建协程池，该协程池的写法比较简单，纯粹为当前项目自用。
// 不支持协程伸缩，不支持任务调度。切不可执行无限循环不自动退出的任务。
// 该协程池就是防止协程数暴增而已。
func NewPool(threads, size int) Executor {
	if threads < 1 {
		threads = runtime.NumCPU() * 2
	}
	if threads < 1 {
		threads = 64
	}
	if size < 1 {
		size = 256
	}

	return &poolExecutor{
		threads: threads,
		runners: make(chan Runner, size),
	}
}

type poolExecutor struct {
	threads int
	runners chan Runner
	once    sync.Once
}

func (pe *poolExecutor) Submit(r Runner) {
	if r == nil {
		return
	}

	pe.once.Do(pe.start)

	pe.runners <- r
}

func (pe *poolExecutor) start() {
	for i := 0; i < pe.threads; i++ {
		go pe.worker()
	}
}

func (pe *poolExecutor) worker() {
	for r := range pe.runners {
		pe.call(r)
	}
}

func (*poolExecutor) call(r Runner) {
	defer func() { recover() }()
	r.Run()
}
