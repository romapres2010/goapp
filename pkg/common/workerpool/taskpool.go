package workerpool

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	_err "github.com/romapres2010/goapp/pkg/common/error"
	_log "github.com/romapres2010/goapp/pkg/common/logger"
)

// TaskPool represent pooling of Task
type TaskPool struct {
	pool sync.Pool
}

// Represent a pool statistics for benchmarking
var (
	countGet uint64 // количество запросов кэша
	countPut uint64 // количество возвратов в кэша
	countNew uint64 // количество создания нового объекта
)

// newTaskPool create new TaskPool
func newTaskPool() *TaskPool {
	p := &TaskPool{
		pool: sync.Pool{
			New: func() interface{} {
				atomic.AddUint64(&countNew, 1)
				task := new(Task)
				task.stopCh = make(chan interface{}, 1)
				task.localDoneCh = make(chan interface{}, 1)
				task.timer = time.NewTimer(POOL_MAX_TIMEOUT)                     // новый таймер - начально максимальное время ожидания
				task.timer.Stop()                                                // остановим таймер, сбрасывать канал не требуется, так как он не сработал
				task.ctx, task.cancel = context.WithCancel(context.Background()) // создаем локальный контекст с отменой
				return task
			},
		},
	}
	return p
}

// getTask allocates a new Task
func (p *TaskPool) getTask() *Task {
	atomic.AddUint64(&countGet, 1)
	task := p.pool.Get().(*Task)
	return task
}

// putTask return Task to pool
func (p *TaskPool) putTask(task *Task) {
	// Если task не был успешно завершен, то в нем могли быть закрыты каналы или сработал таймер - такие не подходят для повторного использования
	if task.state == TASK_STATE_NEW || task.state == TASK_STATE_DONE_SUCCESS {
		atomic.AddUint64(&countPut, 1)
		task.timer.Stop()    // остановим таймер, сбрасывать канал не требуется, так как при TASK_STATE_DONE_SUCCESS он не сработал
		task.requests = nil  // обнулить указатель, чтобы освободить для GC
		task.responses = nil // обнулить указатель, чтобы освободить для GC
		p.pool.Put(task)     // отправить в pool
	}
}

// глобальный TaskPool
var gTaskPool = newTaskPool()

// PrintTaskPoolStats print statistics about task pool
func (p *Pool) PrintTaskPoolStats() {
	if p != nil {
		_log.Info("Usage task pool: countGet, countPut, countNew", countGet, countPut, countNew)
	} else {
		_ = _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, _err.ERR_UNDEFINED_ID, "p != nil").PrintfError()
	}
}
