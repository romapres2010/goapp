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
	TASK_STATE_NEW                    TaskState = iota // task создан
	TASK_STATE_POOL_GET                                // task получен из pool
	TASK_STATE_POOL_PUT                                // task отправлен в pool
	TASK_STATE_READY                                   // task готов к обработкам
	TASK_STATE_IN_PROCESS                              // task выполняется
	TASK_STATE_DONE_SUCCESS                            // task завершился
	TASK_STATE_RECOVER_ERR                             // task остановился из-за паники
	TASK_STATE_TERMINATED_STOP_SIGNAL                  // task остановлен по причине получения сигнала об остановке
	TASK_STATE_TERMINATED_CTX_CLOSED                   // task остановлен по причине закрытия контекста
	TASK_STATE_TERMINATED_TIMEOUT                      // task остановлен по причине превышения timeout
)

// Task - содержимое задачи и результаты выполнения
type Task struct {
	parentCtx context.Context    // родительский контекст, переданный при создании task - используется в функции-обработчике
	ctx       context.Context    // контекст, в рамках которого работает собственно task - используется в функции-обработчике как сигнал для остановки
	cancel    context.CancelFunc // функция закрытия контекста для task

	externalId  uint64             // внешний идентификатор запроса, в рамках которого работает task - для целей логирования
	doneCh      chan<- interface{} // канал сигнала во "внешний мир" о завершении выполнения функции-обработчике
	wg          *sync.WaitGroup    // сигнал во "внешний мир" можно передавать через sync.WaitGroup
	stopCh      chan interface{}   // канал команды на остановку task со стороны "внешнего мира"
	localDoneCh chan interface{}   // локальный канал task - сигнал о завершении выполнения функции-обработчике для "длинных" task

	id        uint64        // номер task - для целей логирования
	state     TaskState     // состояние жизненного цикла task
	prevState TaskState     // предыдущее состояние жизненного цикла task - для контроля перехода статусов
	name      string        // наименование task для логирования и мониторинга
	timeout   time.Duration // максимальное время выполнения для "длинных" task
	timer     *time.Timer   // таймер остановки по timeout для "длинных" task

	requests  []interface{} // входные данные запроса - передаются в функцию-обработчик
	responses []interface{} // результаты обработки запроса в функции-обработчике
	err       error         // ошибки обработки запроса в функции-обработчике

	duration time.Duration // реальная длительность выполнения task

	f func(context.Context, context.Context, ...interface{}) (error, []interface{}) // функция-обработчик

	mx sync.RWMutex
}

func NewTask(parentCtx context.Context, name string, doneCh chan<- interface{}, wg *sync.WaitGroup, id uint64, externalId uint64, timeout time.Duration, f func(context.Context, context.Context, ...interface{}) (error, []interface{}), requests ...interface{}) *Task {
	if f == nil || parentCtx == nil {
		return nil
	}

	// получаем task из глобального кэш
	task := gTaskPool.getTask()

	// Блокировать не требуется, после получения из кэша с ним ни кто не работает
	//task.mx.Lock()
	//defer task.mx.Unlock()
	// Блокировать не требуется, после получения из кэша с ним ни кто не работает

	{ // установим все значения в начальные значения, после получения из кэша
		task.parentCtx = parentCtx

		task.externalId = externalId
		task.doneCh = doneCh
		task.wg = wg

		task.id = id
		task.name = name
		task.timeout = timeout

		task.requests = requests
		task.responses = nil
		task.err = nil

		task.duration = 0

		task.f = f

		task.setStateUnsafe(TASK_STATE_READY) // готов к обработке
	} // установим все значения в начальные значения, после получения из кэша

	return task
}

// DeleteUnsafe - отправить в кэш
func (ts *Task) DeleteUnsafe() {
	if ts != nil {
		gTaskPool.putTask(ts)
	}
}

// setState - установка состояния жизненного цикла task
func (ts *Task) setState(state TaskState) {
	ts.mx.Lock()
	defer ts.mx.Unlock()
	ts.setStateUnsafe(state)
}

// setStateUnsafe - установка состояния жизненного цикла task
func (ts *Task) setStateUnsafe(state TaskState) {
	ts.prevState = ts.state
	ts.state = state
}

// SetDoneChUnsafe - внешний мир может установить канал для уведомления о завершении
func (ts *Task) SetDoneChUnsafe(doneCh chan<- interface{}) {
	ts.doneCh = doneCh
}

// SetWgUnsafe - внешний мир может установить канал для уведомления о завершении
func (ts *Task) SetWgUnsafe(wg *sync.WaitGroup) {
	ts.wg = wg
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
	if ts == nil || ts.f == nil {
		return
	}

	// Заблокируем task на время запуска, чтобы исключить одновременное использование одного указателя
	if ts.mx.TryLock() { // Использование TryLock не рекомендуется, но в данном случае это очень удобно
		defer ts.mx.Unlock()
	} else {
		//_log.Info("TASK - already locked for process: WorkerID, TaskId, TaskExternalId, TaskName", workerID, ts.id, ts.externalId, ts.name)
		ts.err = _err.NewTyped(_err.ERR_WORKER_POOL_TASK_ALREADY_LOCKED, ts.externalId, ts.name, ts.state, ts.prevState).PrintfError()
		return
	}

	// Проверим, что запускать можно только готовый task
	if ts.state == TASK_STATE_READY {
		ts.setStateUnsafe(TASK_STATE_IN_PROCESS)
		//_log.Debug("START task: WorkerID, TaskId, TaskExternalId, TaskName", workerID, ts.id, ts.externalId, ts.name)
	} else {
		//_log.Info("TASK - has incorrect state to run: WorkerID, TaskId, TaskExternalId, TaskName", workerID, ts.id, ts.externalId, ts.name)
		ts.err = _err.NewTyped(_err.ERR_WORKER_POOL_TASK_INCORRECT_STATE, ts.externalId, ts.name, ts.state, ts.prevState, "READY").PrintfError()
		return
	}

	// Обрабатываем панику task
	defer func() {
		if r := recover(); r != nil {
			ts.err = _recover.GetRecoverError(r, ts.externalId, ts.name)
			ts.setStateUnsafe(TASK_STATE_RECOVER_ERR)
		}
	}()

	// Информируем "внешний мир" о завершении работы task в отдельный канал или через wg
	defer func() {
		// Возможна ситуация, когда канал закрыт, например, если "внешний мир" нас не дождался по причине своего таймаута, тогда канал уже будет закрыт
		if ts.doneCh != nil {
			ts.doneCh <- struct{}{}
		}

		// Если работали в рамках WaitGroup, то уменьшим счетчик
		if ts.wg != nil {
			ts.wg.Done()
		}
	}()

	if ts.timeout < 0 {
		// "Короткие" task (timeout < 0) не контролируем по timeout. Их нельзя прервать. Функция-обработчик запускается в goroutine worker
		ts.err, ts.responses = ts.f(ts.parentCtx, nil, ts.requests...)
		ts.setStateUnsafe(TASK_STATE_DONE_SUCCESS)
		return
	} else {
		// "Длинные" task запускаем в фоне и ожидаем завершения в отдельный локальный канал. Контролируем timeout

		var tic = time.Now() // временная метка начала обработки task

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

			// Обработчик получает родительский контекст и локальный контекст task.
			// Локальный контекст task нужно контролировать в обработчике для определения необходимости остановки
			ts.err, ts.responses = ts.f(ts.parentCtx, ts.ctx, ts.requests...)
		}()

		// Определим, нужно ли контролировать timeout, ts.timeout имеет приоритет над workerTimeout
		var timeout time.Duration // предельное время работы task
		if ts.timeout > 0 {
			timeout = ts.timeout
		} else if workerTimeout > 0 {
			timeout = workerTimeout
		}

		// Если timeout == 0, то не контролировать timeout
		if timeout > 0 {
			// Task получает таймер всегда остановленным, сбрасывать канал таймера не требуется, так как он не сработал
			ts.timer.Reset(timeout) // Переставим таймер на новое значение
		}

		// Ожидаем завершения функции обработчика, наступления timeout или команды на закрытие task
		select {
		case <-ts.localDoneCh:
			if !ts.timer.Stop() { // остановим таймер
				<-ts.timer.C // Вероятность, что он сработал в промежутке между select из localDoneCh и выполнением ts.timer.Stop() крайне мала
			}

			ts.duration = time.Now().Sub(tic)
			ts.setStateUnsafe(TASK_STATE_DONE_SUCCESS)
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
			close(ts.localDoneCh)
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
			close(ts.localDoneCh)
			return

			// !!! вариант с time.After(ts.timeout) использовать нельзя, так как будут оставаться "повешенными" таймеры
			//case <-time.After(ts.timeout):
			//	//_log.Info("Task - INTERRUPT - exceeded TaskTimeout: WorkerId, TaskExternalId, TaskName, TaskTimeout", workerID, ts.externalId, ts.name, ts.timeout)
			//	ts.err = _err.NewTyped(_err.ERR_WORKER_POOL_TIMEOUT_ERROR, ts.externalId, ts.timeout).PrintfError()
			//	ts.setState(TASK_STATE_TERMINATED_TIMEOUT)
			//	return
			// !!! вариант с time.After(ts.timeout) использовать нельзя, так как будут оставаться "повешенными" таймеры

			// !!! Контроль за контекстом - дорогая операция
			//case <-ts.ctx.Done():
			//	//_log.Info("Task - STOP - got context close: WorkerId, TaskExternalId", workerID, ts.externalId)
			//	ts.err = _err.NewTyped(_err.ERR_WORKER_POOL_CONTEXT_CLOSED, ts.externalId, "Worker pool - Task - got context close")
			//	ts.setStateUnsafe(TASK_STATE_TERMINATED_CTX_CLOSED)
			//	return
			// !!! Контроль за контекстом - дорогая операция
		}
	}
}

// Stop - принудительная остановка task
func (ts *Task) Stop() {

	// Останавливать можно только в определенных статусах
	if ts.state == TASK_STATE_NEW || ts.state == TASK_STATE_IN_PROCESS || ts.state == TASK_STATE_DONE_SUCCESS {
		//_log.Debug("TASK  - send STOP signal: TaskExternalId, TaskName", ts.externalId, ts.name)
		// Отправляем сигнал и закрываем канал
		if ts.stopCh != nil {
			ts.stopCh <- true
			close(ts.stopCh)
		}
	}
}
