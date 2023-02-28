package workerpool

import (
    "context"
    "fmt"
    "sync"
    "time"

    _err "github.com/romapres2010/goapp/pkg/common/error"
    _log "github.com/romapres2010/goapp/pkg/common/logger"
    _recover "github.com/romapres2010/goapp/pkg/common/recover"
)

// TaskState - статусы жизненного цикла task
type TaskState int

const (
    TASK_STATE_NEW                          TaskState = iota // task создан
    TASK_STATE_PROCESS                                       // task выполняется
    TASK_STATE_DONE                                          // task завершился
    TASK_STATE_RECOVER_ERR                                   // task остановился из-за паники
    TASK_STATE_TERMINATED_STOP_SIGNAL                        // task остановлен по причине получения сигнала об остановке
    TASK_STATE_TERMINATED_PARENT_CTX_CLOSED                  // task остановлен по причине закрытия родительского контекста
    TASK_STATE_TERMINATED_WORKER_CTX_CLOSED                  // task остановлен по причине закрытия worker контекста
    TASK_STATE_TERMINATED_POOL_CTX_CLOSED                    // task остановлен по причине закрытия контекста pool
    TASK_STATE_TERMINATED_TASK_TIMEOUT                       // task остановлен по причине превышения task timeout
    TASK_STATE_TERMINATED_WORKER_TIMEOUT                     // task остановлен по причине превышения worker timeout
)

// Task - содержимое задачи и результаты выполнения
type Task struct {
    parentCtx context.Context    // родительский контекст, переданный при создании task
    ctx       context.Context    // контекст, в рамках которого работает task
    cancel    context.CancelFunc // функция закрытия контекста для task

    externalId uint64             // внешний идентификатор, в рамках которого работает task
    doneCh     chan<- interface{} // канал сигнала о завершении выполнения задачи
    stopCh     chan interface{}   // канал остановки task, запущенного в фоне

    id      uint64        // номер task
    state   TaskState     // состояние жизненного цикла task
    name    string        // наименование task для логирования
    timeout time.Duration // максимальное время выполнения task

    requests  []interface{} // входные данные task
    responses []interface{} // результаты task
    err       error         // ошибки task

    duration time.Duration // реальная длительность выполнения task

    // функция выполнения task
    f func(context.Context, ...interface{}) (error, []interface{})

    mx sync.RWMutex
}

func NewTask(parentCtx context.Context, name string, doneCh chan<- interface{}, id uint64, externalId uint64, timeout time.Duration, f func(context.Context, ...interface{}) (error, []interface{}), requests ...interface{}) *Task {
    if f == nil {
        return nil
    }

    // получаем task из глобального кэш
    task := gTaskPool.getTask()

    task.mx.Lock()
    defer task.mx.Unlock()

    { // установим все значения в начальные значения, после получения из кэша
        task.parentCtx = parentCtx
        task.ctx = nil
        task.cancel = nil

        task.externalId = externalId
        task.doneCh = doneCh
        task.stopCh = nil

        task.id = id
        task.setStateUnsafe(TASK_STATE_NEW)
        task.name = name
        task.timeout = timeout

        task.requests = requests
        task.responses = nil
        task.err = nil

        task.duration = 0

        task.f = f

        if timeout == 0 {
            task.timeout = POOL_DEF_MAX_TIMEOUT
        }
    } // установим все значения в начальные значения, после получения из кэша

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
func (ts *Task) process(poolCtx context.Context, workerCtx context.Context, workerID uint, workerTimeout time.Duration) {
    if ts == nil {
        return
    }

    { // блокируем для проверки и установки статусов
        ts.mx.Lock()

        // запускать можно только новый task, не всегда правильно - создает доп нагрузку на GC
        if ts.state == TASK_STATE_NEW {
            ts.setStateUnsafe(TASK_STATE_PROCESS)
            _log.Debug("START task: WorkerID, TaskId, TaskExternalId, TaskName", workerID, ts.id, ts.externalId, ts.name)
        } else {
            _log.Info("TASK - has incorrect state to run: WorkerID, TaskId, TaskExternalId, TaskName", workerID, ts.id, ts.externalId, ts.name)
            ts.err = _err.NewTyped(_err.ERR_WORKER_POOL_RUN_INCORRECT_STATE, ts.externalId, ts.name, ts.state, "NEW").PrintfError()
        }

        ts.mx.Unlock()

        if ts.err != nil {
            return
        }
    } // блокируем для проверки и установки статусов

    // Работаем в изолированном от родительского контексте
    ts.ctx, ts.cancel = context.WithCancel(context.Background())

    var tic = time.Now()                        // временная метка начала обработки task
    var localDoneCh = make(chan interface{}, 1) // Локальный внутренний канал для информирования о завершении task
    //defer close(localDoneCh) закрывать канал нужно в том месте, где отправляется сигнал

    // канал для информирования task о необходимости срочной остановки
    ts.stopCh = make(chan interface{}, 1)
    //defer close(ts.stopCh) закрывать канал нужно в том месте, где отправляется сигнал

    // информируем о завершении работы и закрываем локальный контекст
    defer func() {
        if r := recover(); r != nil {
            ts.err = _recover.GetRecoverError(r, ts.externalId, ts.name)
            ts.setState(TASK_STATE_RECOVER_ERR)
        }

        // Закрываем локальный контекст task - функция обработчика должна корректно отработать это состояние и выполнить компенсационные воздействия
        if ts.cancel != nil {
            _log.Debug("Task - DONE - close local context: WorkerID, TaskId, TaskExternalId, TaskName", workerID, ts.id, ts.externalId, ts.name)
            ts.cancel()
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
            if localDoneCh != nil {
                localDoneCh <- struct{}{}
                close(localDoneCh)
            }
        }()

        // запускаем обработчик task в локальном контексте task
        if ts.f != nil {
            ts.err, ts.responses = ts.f(ts.ctx, ts.requests...)
        }
    }()

    // максимальное время ожидания, задается на уровне worker
    if workerTimeout == 0 {
        workerTimeout = POOL_DEF_MAX_TIMEOUT
    }

    // Ожидаем завершения обработки, либо таймаутов задачи, worker или закрытия контекста
    select {
    case <-localDoneCh:
        ts.duration = time.Now().Sub(tic)
        ts.setState(TASK_STATE_DONE)
    case <-time.After(ts.timeout):
        _log.Info("Task - INTERRUPT - exceeded TaskTimeout: WorkerId, TaskExternalId, TaskName, TaskTimeout", workerID, ts.externalId, ts.name, ts.timeout)
        ts.err = _err.NewTyped(_err.ERR_WORKER_POOL_TIMEOUT_ERROR, ts.externalId, ts.timeout).PrintfError()
        ts.setState(TASK_STATE_TERMINATED_TASK_TIMEOUT)
    case <-time.After(workerTimeout):
        _log.Info("Task - INTERRUPT - exceeded WorkerTimeout: WorkerId, TaskExternalId, TaskName, WorkerTimeout", workerID, ts.externalId, ts.name, workerTimeout)
        ts.err = _err.NewTyped(_err.ERR_WORKER_POOL_TIMEOUT_ERROR, ts.externalId, workerTimeout).PrintfError()
        ts.setState(TASK_STATE_TERMINATED_WORKER_TIMEOUT)
    case _, ok := <-ts.stopCh:
        if ok {
            // канал был открыт и получили команду на остановку
            _log.Debug("Task - INTERRUPT - got stop signal: WorkerId, TaskExternalId, TaskName, WorkerTimeout", workerID, ts.externalId, ts.name, workerTimeout)
            ts.err = _err.NewTyped(_err.ERR_WORKER_POOL_STOP_SIGNAL, ts.externalId, fmt.Sprintf("[WorkerId='%v', TaskExternalId='%v', TaskName='%v', WorkerTimeout='%v']", workerID, ts.externalId, ts.name, workerTimeout))
            ts.setState(TASK_STATE_TERMINATED_STOP_SIGNAL)
        } else {
            // Не корректная ситуация с внутренней логикой - логируем для анализа
            _log.Error("Task - INTERRUPT - stop chanel closed: WorkerId, TaskExternalId, TaskName, WorkerTimeout", workerID, ts.externalId, ts.name, workerTimeout)
        }
    case <-ts.parentCtx.Done():
        _log.Info("Task - STOP - got Parent context close: WorkerId, TaskExternalId", workerID, ts.externalId)
        ts.err = _err.NewTyped(_err.ERR_WORKER_POOL_CONTEXT_CLOSED, ts.externalId, "Worker pool - Task - got Parent context close")
        ts.setState(TASK_STATE_TERMINATED_PARENT_CTX_CLOSED)
    case <-workerCtx.Done():
        _log.Info("Task - STOP - got WORKER context close: WorkerId, TaskExternalId", workerID, ts.externalId)
        ts.err = _err.NewTyped(_err.ERR_WORKER_POOL_CONTEXT_CLOSED, ts.externalId, "Worker pool - Task - got WORKER context close")
        ts.setState(TASK_STATE_TERMINATED_WORKER_CTX_CLOSED)
    case <-poolCtx.Done():
        _log.Info("Task - STOP - got POOL context close: WorkerId, TaskExternalId", workerID, ts.externalId)
        ts.err = _err.NewTyped(_err.ERR_WORKER_POOL_CONTEXT_CLOSED, ts.externalId, "Worker pool - Task - got POOL context close")
        ts.setState(TASK_STATE_TERMINATED_POOL_CTX_CLOSED)
    }
}

// Stop - принудительная остановка task
func (ts *Task) Stop() {

    ts.mx.Lock()
    defer ts.mx.Unlock()

    // Останавливать можно только в определенных статусах
    if ts.state == TASK_STATE_NEW || ts.state == TASK_STATE_PROCESS || ts.state == TASK_STATE_DONE {
        _log.Debug("TASK  - send STOP signal: TaskExternalId, TaskName", ts.externalId, ts.name)
        // Отправляем сигнал и закрываем канал - если task ни разу не запускался, то ts.stopCh будет nil
        if ts.stopCh != nil {
            ts.stopCh <- true
            close(ts.stopCh)
        }
    }
}
