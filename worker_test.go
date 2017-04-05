package worker_test

import (
	"testing"
	"time"

	"github.com/noypi/worker"
	assertpkg "github.com/stretchr/testify/assert"
	"github.com/twinj/uuid"
)

func TestWorkerPool(t *testing.T) {
	assert := assertpkg.New(t)

	var pool worker.WorkerPool

	type _workinfo struct {
		begin time.Time
		id    string
		end   time.Time
	}
	newWork := func(info *_workinfo) func() {
		return func() {
			info.begin = time.Now()
			info.id = uuid.NewV4().String()
			time.Sleep(300 * time.Millisecond)
			//t.Log("id=", info.id)
			time.Sleep(300 * time.Millisecond)
			info.end = time.Now()
		}
	}

	pool.Start(2)
	works := make([]*_workinfo, 2)
	for i := 0; i < 2; i++ {
		works[i] = new(_workinfo)
		pool.AddWork(newWork(works[i]))
		time.Sleep(200 * time.Millisecond)
	}
	//time.Sleep(600 * time.Millisecond)
	pool.QuitWait()

	assert.NotEqual(works[0].id, works[1].id, "ids should not be equal0")
	//t.Log("begin diff=", works[1].begin.Sub(works[0].begin), ", begin0=", works[0].begin)
	assert.True((150 * time.Millisecond) < works[1].begin.Sub(works[0].begin))
	//t.Log("end diff=", works[1].end.Sub(works[0].end), ",end0=", works[0].end)
	assert.True((150 * time.Millisecond) < works[1].end.Sub(works[0].end))

	// less than 600ms (work time)
	// and more than 200ms in between add work
	assert.True(works[1].end.Sub(works[0].end) < (210 * time.Millisecond))
}
