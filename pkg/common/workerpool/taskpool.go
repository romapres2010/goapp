package workerpool

import (
    "sync"
)

// TaskPool represent pooling of Task
type TaskPool struct {
    pool sync.Pool
}

// newTaskPool create new TaskPool
func newTaskPool() *TaskPool {
    p := &TaskPool{
        pool: sync.Pool{
            New: func() interface{} {
                return new(Task)
            },
        },
    }
    return p
}

// getTask allocates a new Task
func (p *TaskPool) getTask() *Task {
    task := p.pool.Get().(*Task)
    return task
}

// putTask return Task to pool
func (p *TaskPool) putTask(task *Task) {
    p.pool.Put(task)
}

// глобальный TaskPool
var gTaskPool = newTaskPool()
