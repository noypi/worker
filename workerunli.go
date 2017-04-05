package worker

import (
	"sync"
	"time"
)

type WorkerPoolUnlimited struct {
	// max idle time before closing a worker
	MaxIdle time.Duration
	pool    sync.Pool
	workers map[*_workerunli]struct{}
	l       sync.Mutex
}

func (this *WorkerPoolUnlimited) AddWork(task func()) (err error) {
	if nil == this.pool.New {
		this.pool.New = this.newWorker
		this.workers = map[*_workerunli]struct{}{}
	}

	this.l.Lock()
	worker := this.pool.Get().(*_workerunli)
	if nil == worker.task {
		worker.task = make(chan func(), 1)
	}
	if nil == worker.quit {
		worker.quit = make(chan struct{})
	}

	if 0 == this.MaxIdle {
		worker.maxidle = 2 * time.Minute
	} else {
		worker.maxidle = this.MaxIdle
	}

	if !worker.started {
		worker.workerunli = this
		this.workers[worker] = struct{}{}
		go worker.waitForJob()
		worker.started = true

	}
	this.l.Unlock()

	worker.task <- task

	return nil

}

func (this *WorkerPoolUnlimited) QuitWait() {
	this.quit()
}

func (this *WorkerPoolUnlimited) Quit() {
	go this.quit()
}

func (this *WorkerPoolUnlimited) newWorker() interface{} {
	return new(_workerunli)
}

func (this *WorkerPoolUnlimited) putWorker(worker *_workerunli) {
	this.l.Lock()

	close(worker.quit)
	worker.quit = nil

	close(worker.task)
	worker.task = nil

	worker.started = false

	delete(this.workers, worker)
	this.pool.Put(worker)

	this.l.Unlock()
}

func (this *WorkerPoolUnlimited) quit() {
	this.l.Lock()
	for worker, _ := range this.workers {
		worker.quit <- struct{}{}
		close(worker.task)
		close(worker.quit)
	}
	this.l.Unlock()
}

type _workerunli struct {
	task       _taskch
	quit       chan struct{}
	maxidle    time.Duration
	started    bool
	workerunli *WorkerPoolUnlimited
}

func (this *_workerunli) waitForJob() {
	for {
		timeout := time.After(this.maxidle)
		select {
		case job := <-this.task:
			job()

		case <-timeout:
			this.workerunli.putWorker(this)
			return
		case <-this.quit:
			return
		}
	}

}
