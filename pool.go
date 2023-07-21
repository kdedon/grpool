package grpool

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type (
	Runnable interface {
		Run() error
	}

	Errorable interface {
		Handle(error)
	}

	RunnablePanicError struct {
		p any
	}

	Pool struct {
		ctx    context.Context
		cancel context.CancelFunc
		work   chan Runnable
		wg     *sync.WaitGroup
		m      sync.RWMutex
	}

	routine struct {
		ctx context.Context
		w   chan Runnable
		wg  *sync.WaitGroup
	}
)

func newRunnablePanicError(a any) error {
	return &RunnablePanicError{p: a}
}

func (rp RunnablePanicError) Error() string {
	return fmt.Sprintf("panic: %v", rp.p)
}

func (rp RunnablePanicError) Value() any {
	return rp.p
}

func NewPool(ctx context.Context, numRoutines, bufferSize int) *Pool {
	if ctx == nil {
		ctx = context.Background()
	}
	cCtx, cancel := context.WithCancel(ctx)
	p := &Pool{
		ctx:    cCtx,
		cancel: cancel,
		work:   make(chan Runnable, bufferSize),
		wg:     &sync.WaitGroup{},
	}

	for i := 0; i < numRoutines; i++ {
		newRoutine(cCtx, p.work, p.wg)
	}

	return p
}

func (p *Pool) Run(r Runnable) error {
	p.m.RLock()
	defer p.m.RUnlock()

	if p.work != nil {
		p.work <- r
		return nil
	} else {
		return errors.New("work queue closed")
	}
}

func (p *Pool) Close(wait bool) {
	p.m.Lock()
	defer p.m.Unlock()
	close(p.work)
	p.work = nil

	if wait {
		p.wg.Wait()
	}
}

func newRoutine(ctx context.Context, w chan Runnable, wg *sync.WaitGroup) {
	r := &routine{
		ctx: ctx,
		w:   w,
		wg:  wg,
	}
	wg.Add(1)
	go r.loop()
}

func handleError(a any, e error) {
	if o, ok := a.(Errorable); ok {
		o.Handle(e)
	}
}

func doWork(o Runnable) {
	defer func() {
		if a := recover(); a != nil {
			handleError(o, newRunnablePanicError(a))
		}
	}()

	if err := o.Run(); err != nil {
		handleError(o, err)
	}
}

func (r *routine) loop() {
	defer r.wg.Done()

	for {
		select {
		case o, ok := <-r.w:
			if !ok {
				return
			}
			doWork(o)
		case <-r.ctx.Done():
			return
		}
	}
}
