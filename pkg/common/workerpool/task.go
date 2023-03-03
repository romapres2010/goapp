package workerpool

import (
	"context"
	"fmt"
	"sync"
	"time"

	_err "github.com/romapres2010/goapp/pkg/common/error"
	_recover "github.com/romapres2010/goapp/pkg/common/recover"
)

// TaskState - статусы жизненного цикла task
type TaskState int

const (
	TASK_STATE_NEW                          TaskState = iota // task создан
	TASK_STATE_IN_PROCESS                                    // task выполняется
	TASK_STATE_DONE_SUCCESS                                  // task завершился
	TASK_STATE_RECOVER_ERR                                   // task остановился из-за паники
	TASK_STATE_TERMINATED_STOP_SIGNAL                        // task остановлен по причине получения сигнала об остановке
	TASK_STATE_TERMINATED_PARENT_CTX_CLOSED                  // task остановлен по причине закрытия родительского контекста
	TASK_STATE_TERMINATED_WORKER_CTX_CLOSED                  // task остановлен по причине закрытия worker контекста
	TASK_STATE_TERMINATED_POOL_CTX_CLOSED                    // task остановлен по причине закрытия контекста pool
	TASK_STATE_TERMINATED_TIMEOUT                            // task остановлен по причине превышения timeout
)

// Task - содержимое задачи и результаты выполнения
type Task struct {
	parentCtx context.Context    // родительский контекст, переданный при создании task
	ctx       context.Context    // контекст, в рамках которого работает task
	cancel    context.CancelFunc // функция закрытия контекста для task

	externalId  uint64             // внешний идентификатор, в рамках которого работает task
	doneCh      chan<- interface{} // канал сигнала о завершении выполнения задачи
	stopCh      chan interface{}   // канал остановки task, запущенного в фоне
	localDoneCh chan interface{}   // локальный канал task о завершении задачи

	id      uint64        // номер task
	state   TaskState     // состояние жизненного цикла task
	name    string        // наименование task для логирования
	timeout time.Duration // максимальное время выполнения task
	timer   *time.Timer   // таймер остановки по timeout

	requests  []interface{} // входные данные task
	responses []interface{} // результаты task
	err       error         // ошибки task

	duration time.Duration // реальная длительность выполнения task

	// функция выполнения task
	f func(context.Context, context.Context, ...interface{}) (error, []interface{})

	mx sync.RWMutex
}

func NewTask(parentCtx context.Context, name string, doneCh chan<- interface{}, id uint64, externalId uint64, timeout time.Duration, f func(context.Context, context.Context, ...interface{}) (error, []interface{}), requests ...interface{}) *Task {
	if f == nil || parentCtx == nil {
		return nil
	}

	// получаем task из глобального кэш
	task := gTaskPool.getTask()

	task.mx.Lock()
	defer task.mx.Unlock()

	{ // установим все значения в начальные значения, после получения из кэша
		task.parentCtx = parentCtx

		task.externalId = externalId
		task.doneCh = doneCh

		task.id = id
		task.setStateUnsafe(TASK_STATE_NEW)
		task.name = name
		task.timeout = timeout

		task.requests = requests
		task.responses = nil
		task.err = nil

		task.duration = 0

		task.f = f
	} // установим все значения в начальные значения, после получения из кэша

	// !!! Не желательно создавать контест от родительского - это приводит к созданию нового канала для контроля отмены родительского контекста
	//if parentCtx == nil {
	//    task.ctx, task.cancel = context.WithCancel(context.Background())
	//} else {
	//    task.ctx, task.cancel = context.WithCancel(parentCtx)
	//}
	// !!! Не желательно создавать контест от родительского - это приводит к созданию нового канала для контроля отмены родительского контекста

	return task
}

// Delete - отправить в кэш
func (ts *Task) Delete() {
	if ts != nil {
		gTaskPool.putTask(ts)
	}
}

// clearUnsafe - очистка task для повторного использования
func (ts *Task) clearUnsafe() {
}

// setState - установка состояния жизненного цикла task
func (ts *Task) setState(state TaskState) {
	ts.mx.Lock()
	defer ts.mx.Unlock()
	ts.setStateUnsafe(state)
}

// GetState - проверка состояния жизненного цикла task
func (ts *Task) GetState() TaskState {
	ts.mx.RLock()
	defer ts.mx.RUnlock()
	return ts.getStateUnsafe()
}

// getStateUnsafe - проверка состояния жизненного цикла task
func (ts *Task) getStateUnsafe() TaskState {
	return ts.state
}

// setStateUnsafe - установка состояния жизненного цикла task
func (ts *Task) setStateUnsafe(state TaskState) {
	ts.state = state
}

// SetDoneCh - внешний мир может установить канал для уведомления о завершении
func (ts *Task) SetDoneCh(doneCh chan<- interface{}) {
	ts.doneCh = doneCh
}

// GetExternalId - считать externalId из task
func (ts *Task) GetExternalId() uint64 {
	return ts.externalId
}

// GetError - считать ошибки из task
func (ts *Task) GetError() error {
	return ts.err
}

// SetError - установить ошибки из task
func (ts *Task) SetError(err error) {
	ts.err = err
}

// GetResponses - считать результаты из task
func (ts *Task) GetResponses() []interface{} {
	return ts.responses
}

// GetRequests - считать результаты задания task
func (ts *Task) GetRequests() []interface{} {
	return ts.requests
}

// process - запускает task из worker
func (ts *Task) process(workerID uint, workerTimeout time.Duration) {
	if ts == nil {
		return
	}

	// Task можно запустить только один раз
	if ts.mx.TryLock() {
		defer ts.mx.Unlock()
	} else {
		//_log.Info("TASK - already locked for process: WorkerID, TaskId, TaskExternalId, TaskName", workerID, ts.id, ts.externalId, ts.name)
		ts.err = _err.NewTyped(_err.ERR_WORKER_POOL_TASK_ALREADY_LOCKED, ts.externalId, ts.name, ts.state).PrintfError()
		return
	}

	// запускать можно только новый task
	if ts.state == TASK_STATE_NEW {
		ts.setStateUnsafe(TASK_STATE_IN_PROCESS)
		//_log.Debug("START task: WorkerID, TaskId, TaskExternalId, TaskName", workerID, ts.id, ts.externalId, ts.name)
	} else {
		//_log.Info("TASK - has incorrect state to run: WorkerID, TaskId, TaskExternalId, TaskName", workerID, ts.id, ts.externalId, ts.name)
		ts.err = _err.NewTyped(_err.ERR_WORKER_POOL_RUN_INCORRECT_STATE, ts.externalId, ts.name, ts.state, "NEW").PrintfError()
		return
	}

	var tic = time.Now()      // временная метка начала обработки task
	var timeout time.Duration // предельное время работы task

	// информируем о завершении работы и закрываем локальный контекст
	defer func() {
		if r := recover(); r != nil {
			ts.err = _recover.GetRecoverError(r, ts.externalId, ts.name)
			ts.setStateUnsafe(TASK_STATE_RECOVER_ERR)
		}

		// информируем внешний мир о завершении task - общий канал для группировки связанных task
		if ts.doneCh != nil {
			ts.doneCh <- struct{}{}
		}
	}()

	// Запускаем task в фоне и ожидаем завершения в локальный канал или timeout
	go func() {
		defer func() {
			if r := recover(); r != nil {
				ts.err = _recover.GetRecoverError(r, ts.externalId, ts.name)
			}

			// Отправляем сигнал и закрываем канал, task не контролирует, успешно или нет завершился обработчик
			if ts.localDoneCh != nil {
				ts.localDoneCh <- struct{}{}
			}
		}()

		// запускаем обработчик task в локальном контексте task
		if ts.f != nil {
			ts.err, ts.responses = ts.f(ts.parentCtx, ts.ctx, ts.requests...)
		}
	}()

	{ // определим, нужно ли контролировать timeout
		// наименьший из workerTimeout и ts.timeout
		if workerTimeout < ts.timeout {
			timeout = workerTimeout
		} else {
			timeout = ts.timeout
		}

		if timeout > 0 {
			// Переставим таймер на новое значение
			// Task получает таймер всегда остановленным, сбрасывать канал таймера не требуется, так как он не сработал
			//ts.timer.Stop()
			ts.timer.Reset(timeout)
		}
	}

	// Ожидаем завершения обработки, либо таймаутов задачи, worker или закрытия контекста
	select {
	case <-ts.localDoneCh:
		ts.duration = time.Now().Sub(tic)
		ts.setStateUnsafe(TASK_STATE_DONE_SUCCESS)
		ts.timer.Stop() // остановим таймер, сбрасывать канал не требуется, так как он не сработал
		//_log.Debug("Task - DONE: WorkerID, TaskId, TaskExternalId, TaskName", workerID, ts.id, ts.externalId, ts.name)
		return
	case _, ok := <-ts.stopCh:
		if ok {
			// канал был открыт и получили команду на остановку
			//_log.Debug("Task - INTERRUPT - got stop signal: WorkerId, TaskExternalId, TaskName, WorkerTimeout", workerID, ts.externalId, ts.name, workerTimeout)
			ts.err = _err.NewTyped(_err.ERR_WORKER_POOL_STOP_SIGNAL, ts.externalId, fmt.Sprintf("[WorkerId='%v', TaskExternalId='%v', TaskName='%v', WorkerTimeout='%v']", workerID, ts.externalId, ts.name, workerTimeout))
			ts.setStateUnsafe(TASK_STATE_TERMINATED_STOP_SIGNAL)
		} else {
			// Не корректная ситуация с внутренней логикой - логируем для анализа
			//_log.Error("Task - INTERRUPT - stop chanel closed: WorkerId, TaskExternalId, TaskName, WorkerTimeout", workerID, ts.externalId, ts.name, workerTimeout)
		}
		// Закрываем локальный контекст task - функция обработчика должна корректно отработать это состояние и выполнить компенсационные воздействия
		if ts.cancel != nil {
			//_log.Debug("Task - close local context: WorkerID, TaskId, TaskExternalId, TaskName", workerID, ts.id, ts.externalId, ts.name)
			ts.cancel()
		}
		return
	case <-ts.timer.C:
		//_log.Info("Task - INTERRUPT - exceeded Timeout: WorkerId, TaskExternalId, TaskName, Timeout", workerID, ts.externalId, ts.name, timeout)
		ts.err = _err.NewTyped(_err.ERR_WORKER_POOL_TIMEOUT_ERROR, ts.externalId, ts.id, timeout).PrintfError()
		ts.setStateUnsafe(TASK_STATE_TERMINATED_TIMEOUT)
		// Закрываем локальный контекст task - функция обработчика должна корректно отработать это состояние и выполнить компенсационные воздействия
		if ts.cancel != nil {
			//_log.Debug("Task - close local context: WorkerID, TaskId, TaskExternalId, TaskName", workerID, ts.id, ts.externalId, ts.name)
			ts.cancel()
		}
		return

		// !!! вариант с time.After(ts.timeout) использовать нельзя, так как будут оставаться "повешенными" таймеры, пока они не сработают
		//case <-time.After(ts.timeout):
		//	//_log.Info("Task - INTERRUPT - exceeded TaskTimeout: WorkerId, TaskExternalId, TaskName, TaskTimeout", workerID, ts.externalId, ts.name, ts.timeout)
		//	ts.err = _err.NewTyped(_err.ERR_WORKER_POOL_TIMEOUT_ERROR, ts.externalId, ts.timeout).PrintfError()
		//	ts.setState(TASK_STATE_TERMINATED_TIMEOUT)
		//	return
		// !!! вариант с time.After(ts.timeout) использовать нельзя, так как будут оставаться "повешенными" таймеры, пока они не сработают

		// !!! Контроль за контекстом приводит к необходимости блокировки родительского канала контекста - дорогая операция
		//case <-ts.parentCtx.Done():
		//	//_log.Info("Task - STOP - got context close: WorkerId, TaskExternalId", workerID, ts.externalId)
		//	ts.err = _err.NewTyped(_err.ERR_WORKER_POOL_CONTEXT_CLOSED, ts.externalId, "Worker pool - Task - got context close")
		//	ts.setStateUnsafe(TASK_STATE_TERMINATED_PARENT_CTX_CLOSED)
		//	return
		// !!! Контроль за контекстом приводит к необходимости блокировки родительского канала контекста - дорогая операция

		// !!! Контроль за контекстом приводит к необходимости блокировки родительского канала контекста - дорогая операция
		//case <-ts.ctx.Done():
		//	//_log.Info("Task - STOP - got context close: WorkerId, TaskExternalId", workerID, ts.externalId)
		//	ts.err = _err.NewTyped(_err.ERR_WORKER_POOL_CONTEXT_CLOSED, ts.externalId, "Worker pool - Task - got context close")
		//	ts.setStateUnsafe(TASK_STATE_TERMINATED_PARENT_CTX_CLOSED)
		//	return
		// !!! Контроль за контекстом приводит к необходимости блокировки родительского канала контекста - дорогая операция
	}
}

// Stop - принудительная остановка task
func (ts *Task) Stop() {

	// Останавливать можно только в определенных статусах
	if ts.state == TASK_STATE_NEW || ts.state == TASK_STATE_IN_PROCESS || ts.state == TASK_STATE_DONE_SUCCESS {
		//_log.Debug("TASK  - send STOP signal: TaskExternalId, TaskName", ts.externalId, ts.name)
		// Отправляем сигнал и закрываем канал - если task ни разу не запускался, то ts.stopCh будет nil
		if ts.stopCh != nil {
			ts.stopCh <- true
			close(ts.stopCh)
		}
	}
}
