package worker

import (
	"context"
	"time"
)

type Pool struct {
	workers chan struct{}
	tasks   chan Task
	quit    chan struct{}
}

type Task struct {
	Ctx      context.Context
	Work     func() error
	Priority int
}

func NewPool(maxWorkers int) *Pool {
	p := &Pool{
		workers: make(chan struct{}, maxWorkers),
		tasks:   make(chan Task, 100),
		quit:    make(chan struct{}),
	}
	
	go p.dispatcher()
	return p
}

func (p *Pool) dispatcher() {
	for {
		select {
		case <-p.quit:
			return
		case task := <-p.tasks:
			select {
			case p.workers <- struct{}{}:
				go func() {
					defer func() { <-p.workers }()
					
					done := make(chan error, 1)
					go func() {
						done <- task.Work()
					}()

					select {
					case <-task.Ctx.Done():
						return
					case <-done:
						return
					case <-time.After(10 * time.Second):
						return
					}
				}()
			default:
				go func() {
					time.Sleep(100 * time.Millisecond)
					p.Submit(task)
				}()
			}
		}
	}
}

func (p *Pool) Submit(task Task) {
	select {
	case p.tasks <- task:
	default:
		go func() {
			time.Sleep(100 * time.Millisecond)
			p.Submit(task)
		}()
	}
}

func (p *Pool) Shutdown() {
	close(p.quit)
}
