package yao

import (
	"fmt"
	"time"

	"rogchap.com/v8go"
)

// Push add *v8go.Context to pool
func (p *Pool) Push(ctx *v8go.Context) {
	select {
	case p.contexts <- ctx:
	default:
		// panic("Queue full")
	}
}

// Pop get *v8go.Context from pool
func (p *Pool) Pop() (*v8go.Context, error) {
	select {
	case ctx := <-p.contexts:
		return ctx, nil
	default:
		return nil, fmt.Errorf("context not enough")
	}
}

// Make get *v8go.Context from pool
func (p *Pool) Make(makeContext func() *v8go.Context) (*v8go.Context, error) {
	p.Push(makeContext())
	return p.Pop()
}

// PushMulti 添加功能多
func (p *Pool) PushMulti(size int, makeContext func() *v8go.Context) {
	for i := 0; i < size; i++ {
		fmt.Println("PushMulti: i", i, size)
		go p.Push(makeContext())
	}
}

// Prepare prepare *v8go.Context to pool
func (p *Pool) Prepare(chunk int, makeContext func() *v8go.Context) {
	for {
		cost := p.size - len(p.contexts)
		p.PushMulti(cost, makeContext)
		time.Sleep(200 * time.Microsecond)
	}
}

// NewPool make a new pool
func NewPool(size int) *Pool {
	p := &Pool{
		contexts: make(chan *v8go.Context, size),
		size:     size,
		lock:     false,
	}
	return p
}
