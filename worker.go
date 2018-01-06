package worker

import (
	"fmt"
	"sync/atomic"
)

type _taskch chan func()
type _poolch chan _taskch

type WorkerPool struct {
	pool        _poolch
	workers     []*_worker
	started     bool
	nActiveWork uint64
}

type _worker struct {
	task        _taskch
	pool        _poolch
	quit        chan struct{}
	pActiveWork *uint64
}

func (this *WorkerPool) Start(maxWorkers int) {
	this.started = true
	this.pool = make(_poolch, maxWorkers)
	this.workers = make([]*_worker, maxWorkers)
	for i := 0; i < maxWorkers; i++ {
		worker := new(_worker)
		worker.pool = this.pool
		worker.task = make(chan func(), 1)
		worker.quit = make(chan struct{})
		worker.pActiveWork = &this.nActiveWork
		this.workers[i] = worker
		go worker.waitForJob()
	}
}

func (this *WorkerPool) QuitWait() {
	this.quit()
}

func (this *WorkerPool) Quit() {
	go this.quit()
}

func (this *WorkerPool) quit() {
	for _, worker := range this.workers {
		worker.quit <- struct{}{}
		close(worker.task)
		close(worker.quit)
	}
	close(this.pool)
}

func (this *WorkerPool) AddWork(task func()) (err error) {
	if !this.started {
		return fmt.Errorf("not started.")
	}
	go func() {
		taskch := <-this.pool
		taskch <- task
	}()

	return nil
}

func (this *WorkerPool) ActiveWorksCount() int {
	return int(atomic.LoadUint64(&this.nActiveWork))
}

func (this *_worker) waitForJob() {
	for {
		this.pool <- this.task

		select {
		case job := <-this.task:
			atomic.AddUint64(this.pActiveWork, 1)
			job()
			atomic.AddUint64(this.pActiveWork, ^uint64(0))
		case <-this.quit:
			return
		}
	}

}
