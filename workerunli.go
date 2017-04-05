package worker

import (
	"sync"
	"time"
)

type WorkerPoolUnlimited struct {
	// max idle time before closing a worker
	MaxIdle time.Duration
	pool    sync.Pool
	queue   map[*_workerunli]struct{}
	workers map[*_workerunli]struct{}
	l       sync.Mutex
	lq      sync.Mutex
}

func (this *WorkerPoolUnlimited) AddWork(task func()) (err error) {
	if nil == this.pool.New {
		this.pool.New = this.newWorker
		this.workers = map[*_workerunli]struct{}{}
	}

	var worker *_workerunli

	this.lq.Lock()
	if 0 < len(this.queue) {
		for worker, _ = range this.queue {
			break
		}
		delete(this.queue, worker)

	} else {
		if nil == this.queue {
			this.queue = map[*_workerunli]struct{}{}
		}
		worker = this.pool.Get().(*_workerunli)
	}
	this.lq.Unlock()

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
		this.l.Lock()
		this.workers[worker] = struct{}{}
		this.l.Unlock()

		go worker.waitForJob()
		worker.started = true

	}

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

func (this *WorkerPoolUnlimited) restingWorker(worker *_workerunli) {
	this.lq.Lock()
	// if quit is not yet triggered
	if nil != this.queue {
		this.queue[worker] = struct{}{}
	}
	this.lq.Unlock()
}

func (this *WorkerPoolUnlimited) putWorker(worker *_workerunli) {
	this.l.Lock()
	delete(this.workers, worker)
	this.l.Unlock()

	this.lq.Lock()
	if nil != this.queue {
		delete(this.queue, worker)
	}
	this.lq.Unlock()

	close(worker.quit)
	worker.quit = nil

	close(worker.task)
	worker.task = nil

	worker.started = false
	this.pool.Put(worker)

}

func (this *WorkerPoolUnlimited) quit() {
	this.lq.Lock()
	this.queue = nil
	this.lq.Unlock()

	this.l.Lock()
	for worker, _ := range this.workers {
		worker.quit <- struct{}{}
		close(worker.task)
		worker.task = nil
		close(worker.quit)
		worker.quit = nil
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
			this.workerunli.restingWorker(this)

		case <-timeout:
			this.workerunli.putWorker(this)
			return

		case <-this.quit:
			return
		}
	}

}
