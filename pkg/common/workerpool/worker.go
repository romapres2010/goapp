package workerpool

import (
	"context"
	"sync"
	"time"

	_err "github.com/romapres2010/goapp/pkg/common/error"
	_log "github.com/romapres2010/goapp/pkg/common/logger"
	_metrics "github.com/romapres2010/goapp/pkg/common/metrics"
	_recover "github.com/romapres2010/goapp/pkg/common/recover"
)

// WorkerState - статусы жизненного цикла worker
type WorkerState int

const (
	WORKER_STATE_NEW                           WorkerState = iota // worker создан
	WORKER_STATE_WORKING                                          // worker обрабатывает задачу
	WORKER_STATE_IDLE                                             // worker простаивает
	WORKER_STATE_TERMINATING_PARENT_CTX_CLOSED                    // worker останавливается по причине закрытия родительского контекста
	WORKER_STATE_TERMINATING_STOP_SIGNAL                          // worker останавливается по причине получения сигнала об остановке
	WORKER_STATE_TERMINATING_TASK_CH_CLOSED                       // worker останавливается по причине закрытия канала задач
	WORKER_STATE_TERMINATED                                       // worker остановлен
	WORKER_STATE_RECOVER_ERR                                      // worker остановился из-за паники
)

// Worker - выполняет task
type Worker struct {
	pool *Pool // pool, в состав которого входит worker

	parentCtx context.Context // родительский контекст pool, в котором работает worker

	externalId uint64           // внешний идентификатор, в рамках которого работает worker - для целей логирования
	stopCh     chan interface{} // канал команды на остановку worker со стороны "внешнего мира"

	id      uint                // номер worker - для целей логирования
	state   WorkerState         // состояние жизненного цикла worker
	errCh   chan<- *WorkerError // канал информирования о критичных ошибках worker в pool
	timeout time.Duration       // максимально время ожидания выполнения task, передается в task при запуске

	taskQueueCh   <-chan *Task // канал очереди task
	taskInProcess *Task        // текущая обрабатываемая task

	mx sync.RWMutex
}

// setState - установка состояния жизненного цикла worker
func (wr *Worker) setState(state WorkerState) {
	wr.mx.Lock()
	defer wr.mx.Unlock()
	wr.setStateUnsafe(state)
}

// setStateUnsafe - установка состояния жизненного цикла worker
func (wr *Worker) setStateUnsafe(state WorkerState) {
	wr.state = state
}

// GetState - проверка состояния жизненного цикла worker
func (wr *Worker) GetState() WorkerState {
	wr.mx.RLock()
	defer wr.mx.RUnlock()
	return wr.getStateUnsafe()
}

// getStateUnsafe - проверка состояния жизненного цикла worker
func (wr *Worker) getStateUnsafe() WorkerState {
	return wr.state
}

// newWorker - возвращает новый экземпляр worker-а
func newWorker(parentCtx context.Context, pool *Pool, taskQueueCh <-chan *Task, id uint, externalId uint64, errCh chan<- *WorkerError, timeout time.Duration) *Worker {
	var worker = Worker{
		parentCtx:   parentCtx,
		id:          id,
		pool:        pool,
		externalId:  externalId,
		taskQueueCh: taskQueueCh,
		timeout:     timeout,
		errCh:       errCh,
	}

	worker.setState(WORKER_STATE_NEW)

	return &worker
}

// run - запускает worker
func (wr *Worker) run(wg *sync.WaitGroup) {

	// Заблокируем worker на время запуска, чтобы исключить одновременное использование одного указателя
	if wr.mx.TryLock() { // Использование TryLock не рекомендуется, но в данном случае это очень удобно
		defer wr.mx.Unlock()
	} else {
		//_log.Info("Worker - already locked for run: PoolName, WorkerId, WorkerExternalId, State", wr.pool.name, wr.id, wr.externalId, wr.state)
		err := _err.NewTyped(_err.ERR_WORKER_POOL_ALREADY_LOCKED, wr.externalId, wr.id, wr.state).PrintfError()

		wr.errCh <- &WorkerError{ // ошибки отправляем в общий канал ошибок pool
			err:    err,
			worker: wr,
		}
		return
	}

	// запускать можно только новый worker или после паники
	if wr.state == WORKER_STATE_NEW || wr.state == WORKER_STATE_RECOVER_ERR {
		wr.setStateUnsafe(WORKER_STATE_IDLE) // worker запущен и простаивает
		//_log.Debug("Worker - START: PoolName, WorkerId, WorkerExternalId, State", wr.pool.name, wr.id, wr.externalId, wr.state)
	} else {
		//_log.Info("Worker - has incorrect state to run: PoolName, WorkerId, WorkerExternalId, State", wr.pool.name, wr.id, wr.externalId, wr.state)
		err := _err.NewTyped(_err.ERR_WORKER_POOL_RUN_INCORRECT_STATE, wr.externalId, wr.id, wr.state, "NEW', 'RECOVER_ERR").PrintfError()

		wr.errCh <- &WorkerError{ // ошибки отправляем в общий канал ошибок pool
			err:    err,
			worker: wr,
		}
		return
	}

	// Создаем внутренний канал для информирования worker о необходимости срочной остановки со стороны "внешнего мира"
	wr.stopCh = make(chan interface{}, 1)
	//defer close(wr.stopCh) закрывать канал будем в том месте, где отправляется сигнал

	// Обрабатываем панику worker, если работали в рамках WaitGroup, то уменьшим счетчик
	defer func() {
		if r := recover(); r != nil {
			//_log.Info("Worker - RECOVER FROM PANIC: PoolName, WorkerId, WorkerExternalId, State", wr.pool.name, wr.id, wr.externalId, wr.state)
			err := _recover.GetRecoverError(r, wr.externalId)
			if err != nil {
				wr.setStateUnsafe(WORKER_STATE_RECOVER_ERR)

				wr.errCh <- &WorkerError{ // ошибки отправляем в общий канал ошибок pool
					err:    err,
					worker: wr,
				}
			}
		} else {
			wr.setStateUnsafe(WORKER_STATE_TERMINATED)
		}

		// Если работали в рамках WaitGroup, то уменьшим счетчик
		if wg != nil {
			wg.Done()
		}
	}()

	// Ждем task из канала-очереди taskQueueCh (пустые задачи игнорируем), сигнала об остановки или закрытия родительского контекста pool
	for {
		// Для команды select нет гарантии, что каналы будут опрощены именно в той последовательности, в которой они написаны.
		// Поэтому в каждой новой итерации сначала проверяем, что worker не остановлен
		select {
		case _, ok := <-wr.stopCh:
			if ok { // канал был открыт и получили команду на остановку
				//_log.Info("Worker - STOP - got quit signal: PoolName, WorkerId, WorkerExternalId", wr.pool.name, wr.id, wr.externalId)
				wr.setStateUnsafe(WORKER_STATE_TERMINATING_STOP_SIGNAL)
			} else {
				// Не корректная ситуация с внутренней логикой - логируем для анализа
				_log.Error("Worker - STOP - stop chanel closed: PoolName, WorkerId, WorkerExternalId", wr.pool.name, wr.id, wr.externalId)
			}
			return
		case <-wr.parentCtx.Done():
			// закрыт родительский контекст
			//_log.Info("Worker - STOP - got parent context close: PoolName, WorkerId, WorkerExternalId", wr.pool.name, wr.id, wr.externalId)
			wr.setStateUnsafe(WORKER_STATE_TERMINATING_PARENT_CTX_CLOSED)
			return
		default:
		}

		// Если worker не остановлен, то проверяем канал-очереди задач
		select {
		case task, ok := <-wr.taskQueueCh:
			if ok { // канал очереди задач открыт
				if task != nil { // игнорируем пустые задачи
					//_log.Debug("Worker - got task: PoolName, WorkerId, WorkerExternalId, TaskName", wr.pool.name, wr.id, wr.externalId, task.name)
					_metrics.IncWPWorkerProcessCountVec(wr.pool.name)                               // Метрика - количество worker в работе
					_metrics.SetWPTaskQueueBufferLenVec(wr.pool.name, float64(len(wr.taskQueueCh))) // Метрика - длина необработанной очереди задач

					wr.setStateUnsafe(WORKER_STATE_WORKING)
					wr.taskInProcess = task

					//_log.Debug("Worker - start to process task: PoolName, WorkerId, WorkerExternalId, TaskName", wr.pool.name, wr.id, wr.externalId, task.name)
					task.process(wr.id, wr.timeout)

					wr.taskInProcess = nil
					wr.setStateUnsafe(WORKER_STATE_IDLE)

					_metrics.DecWPWorkerProcessCountVec(wr.pool.name)                            // Метрика - количество worker в работе
					_metrics.IncWPTaskProcessDurationVec(wr.pool.name, task.name, task.duration) // Метрика - время выполнения задачи по имени
					//_log.Debug("Worker - end process task: PoolName, WorkerId, WorkerExternalId, TaskName, Duration", wr.pool.name, wr.id, wr.externalId, task.name, task.duration)
				}
			} else { // Если канал-очереди task закрыт - прерываем работу
				//_log.Debug("Worker - task channel was closed: PoolName, WorkerId, WorkerExternalId", wr.pool.name, wr.id, wr.externalId)
				wr.setStateUnsafe(WORKER_STATE_TERMINATING_TASK_CH_CLOSED)
				return
			}
		case _, ok := <-wr.stopCh:
			if ok { // канал был открыт и получили команду на остановку
				//_log.Info("Worker - STOP - got quit signal: PoolName, WorkerId, WorkerExternalId", wr.pool.name, wr.id, wr.externalId)
				wr.setStateUnsafe(WORKER_STATE_TERMINATING_STOP_SIGNAL)
			} else {
				// Не корректная ситуация с внутренней логикой - логируем для анализа
				_log.Error("Worker - STOP - stop chanel closed: PoolName, WorkerId, WorkerExternalId", wr.pool.name, wr.id, wr.externalId)
			}
			return
		case <-wr.parentCtx.Done():
			// закрыт родительский контекст
			//_log.Info("Worker - STOP - got parent context close: PoolName, WorkerId, WorkerExternalId", wr.pool.name, wr.id, wr.externalId)
			wr.setStateUnsafe(WORKER_STATE_TERMINATING_PARENT_CTX_CLOSED)
			return
		}
	}
}

// Stop - принудительная остановка worker, не дожидаясь отработки всей очереди
func (wr *Worker) Stop(shutdownMode PoolShutdownMode) {
	if wr == nil {
		return
	}

	//_log.Debug("Worker - STOP: PoolName, WorkerId, WorkerExternalId, State", wr.pool.name, wr.id, wr.externalId, wr.state)

	// Останавливать можно только в определенных статусах
	if wr.state == WORKER_STATE_NEW || wr.state == WORKER_STATE_WORKING || wr.state == WORKER_STATE_IDLE {
		//_log.Debug("Worker  - send STOP signal: PoolName, WorkerId", wr.pool.name, wr.id)

		// Отправляем сигнал и закрываем канал - если worker ни разу не запускался, то wr.stopCh будет nil
		if wr.stopCh != nil {
			wr.stopCh <- true
			close(wr.stopCh)
		}

		// В режиме "Hard" остановки запускаем прерывание текущей task
		if shutdownMode == POOL_SHUTDOWN_HARD {
			if wr.taskInProcess != nil {
				wr.taskInProcess.Stop()
			}
		}
	}
}
