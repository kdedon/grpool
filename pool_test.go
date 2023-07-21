package grpool

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

type (
	adder struct {
		i atomic.Int32
	}

	errorer struct {
		result bool
	}

	panicer struct {
		result bool
	}
)

func (a *adder) Run() error {
	a.i.Add(1)
	return nil
}

var ErrorErr = errors.New("error")

func (e *errorer) Run() error {
	return ErrorErr
}

func (e *errorer) Handle(err error) {
	if err == ErrorErr {
		e.result = true
	}
}

func (p *panicer) Run() error {
	panic("panic")
}

func (p *panicer) Handle(err error) {
	var rpe *RunnablePanicError
	if errors.As(err, &rpe) {
		if r, ok := rpe.Value().(string); ok {
			if r == "panic" {
				p.result = true
			}
		}
	}
}

func TestNewPoolBasic(t *testing.T) {
	expected := 5
	p := NewPool(context.Background(), 1, expected)

	a := &adder{}

	for i := 0; i < expected; i++ {
		p.Run(a)
	}
	p.Close(true)

	res := a.i.Load()
	if res != int32(expected) {
		t.Errorf("got %d, expected %d", res, expected)
	}
}

func TestNewPoolAdvanced(t *testing.T) {
	expected := 1000
	p := NewPool(context.Background(), 10, expected)

	a := &adder{}

	for i := 0; i < expected; i++ {
		p.Run(a)
	}
	p.Close(true)

	res := a.i.Load()
	if res != int32(expected) {
		t.Errorf("got %d, expected %d", res, expected)
	}
}

func TestErrorOnClosedChannel(t *testing.T) {
	p := NewPool(context.Background(), 1, 1)

	a := &adder{}

	p.Close(true)
	if err := p.Run(a); err == nil {
		t.Error("expected error on Run()")
	}
}

func TestHandleError(t *testing.T) {
	p := NewPool(context.Background(), 1, 1)

	e := &errorer{}
	p.Run(e)
	p.Close(true)

	if !e.result {
		t.Error("result not set")
	}
}

func TestHandlePanic(t *testing.T) {
	p := NewPool(context.Background(), 1, 1)

	e := &panicer{}
	p.Run(e)
	p.Close(true)

	if !e.result {
		t.Error("result not set")
	}
}
