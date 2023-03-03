package workerpool

import (
	"context"
	"runtime"
	"sync"
	"time"

	_err "github.com/romapres2010/goapp/pkg/common/error"
	_metrics "github.com/romapres2010/goapp/pkg/common/metrics"
	_recover "github.com/romapres2010/goapp/pkg/common/recover"
)

const POOL_MAX_TIMEOUT = time.Hour * 24 * 365

// PoolState - статусы жизненного цикла pool
type PoolState int

// TODO - переделать
const (
	POOL_STATE_NEW               PoolState = iota // pool создан, еще ни разу не запускался
	POOL_STATE_ONLINE_RUNNING                     // pool запущен в режиме online, добавление новых задач запрещено
	POOL_STATE_ONLINE_DONE                        // pool запущенный в режиме online, завершил обработку всех задач
	POOL_STATE_INCOMPLETE_DONE                    // pool запущенный в режиме online, завершил обработку НЕ всех задач
	POOL_STATE_RECOVER_ERR                        // pool остановлен по панике, дальнейшие действия не возможны
	POOL_STATE_BG_RUNNING                         // pool запущен в режиме background, добавление новых задач разрешено
	POOL_STATE_BG_PAUSED                          // обработка worker поставлена на паузу
	POOL_STATE_SHUTTING_DOWN                      // pool находится в режиме остановки, добавление новых задач запрещено
	POOL_STATE_TERMINATE_TIMEOUT                  // pool превышено время ожидания остановки
	POOL_STATE_SHUTDOWN                           // pool экстренно прерван
)

// Config - конфигурационные настройки pool
type Config struct {
	TaskQueueSize     int           `yaml:"task_queue_size" json:"task_queue_size"`       // размер очереди задач - если 0, то количество ядер х 200
	TaskTimeout       time.Duration `yaml:"task_timeout" json:"task_timeout"`             // максимальное время обработки одного расчета
	WorkerConcurrency int           `yaml:"worker_concurrency" json:"worker_concurrency"` // уровень параллелизма - если 0, то количество ядер х 2
	WorkerTimeout     time.Duration `yaml:"worker_timeout" json:"worker_timeout"`         // максимальное время обработки задачи worker
}

// Pool - управление набором worker и выполнения task
type Pool struct {
	cfg *Config // конфиг pool

	parentCtx context.Context    // родительский контекст, в котором создали pool
	ctx       context.Context    // контекст, в котором работает pool
	cancel    context.CancelFunc // функция закрытия контекста для pool

	externalId uint64           // внешний идентификатор для логирования
	name       string           // имя pool для сбора метрик и логирования
	state      PoolState        // состояние жизненного цикла pool
	stopCh     chan interface{} // канал остановки pool, запущенного в фоне

	workers           map[int]*Worker   // набор worker
	workerConcurrency int               // уровень параллелизма - если 0, то количество ядер х 2
	workerTimeout     time.Duration     // таймаут выполнения задачи одним worker
	workerErrCh       chan *WorkerError // канал ошибок workers, размер определяется количеством worker

	taskQueueCh   chan *Task // канал очереди задач, ожидающих выполнения
	taskQueueSize int        // размер очереди задач - если 0, то количество ядер х 200

	mx sync.RWMutex
}

// WorkerError - ошибки и сбойный worker
type WorkerError struct {
	err    error
	worker *Worker
}

// setState - установка состояния жизненного цикла pool
func (p *Pool) setState(state PoolState) {
	p.mx.Lock()
	defer p.mx.Unlock()
	p.setStateUnsafe(state)
}

// setStateUnsafe - установка состояния жизненного цикла pool
func (p *Pool) setStateUnsafe(state PoolState) {
	p.state = state
}

// GetState - проверка состояния жизненного цикла pool
func (p *Pool) GetState() PoolState {
	p.mx.RLock()
	defer p.mx.RUnlock()
	return p.getStateUnsafe()
}

// getStateUnsafe - проверка состояния жизненного цикла pool
func (p *Pool) getStateUnsafe() PoolState {
	return p.state
}

// NewPool инициализирует новый пул
func NewPool(parentCtx context.Context, externalId uint64, name string, cfg *Config) (*Pool, error) {
	//_log.Debug("Pool - create new: ExternalId, PoolName", externalId, name)

	if parentCtx == nil {
		return nil, _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, externalId, "Nil parent context pointer").PrintfError()
	}
	if cfg == nil {
		return nil, _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, externalId, "Nil config pointer").PrintfError()
	}

	var pool = Pool{
		parentCtx:         parentCtx,
		externalId:        externalId,
		name:              name,
		cfg:               cfg,
		workerConcurrency: cfg.WorkerConcurrency,
		taskQueueSize:     cfg.TaskQueueSize,
		workerTimeout:     cfg.WorkerTimeout,
	}

	pool.setState(POOL_STATE_NEW)

	if pool.workerConcurrency == 0 {
		pool.workerConcurrency = runtime.NumCPU() * 2
		//_log.Debug("Set default: ExternalId, PoolName, WorkerConcurrency", externalId, pool.name, pool.workerConcurrency)
	}

	if pool.taskQueueSize == 0 {
		pool.taskQueueSize = pool.workerConcurrency * 10000
		//_log.Debug("Set default: ExternalId, PoolName, TaskQueueSize", externalId, pool.name, pool.taskQueueSize)
	}

	return &pool, nil
}

// AddTask добавляет task в pool
func (p *Pool) AddTask(task *Task) (err error) {
	if p == nil {
		return _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, _err.ERR_UNDEFINED_ID, "Nil Pool pointer").PrintfError()
	}

	// Пустую задачу игнорируем
	if task == nil {
		return nil
	}

	//_log.Debug("Pool - START - add new task: PoolName, TaskId, TaskExternalId, State", p.name, task.id, task.externalId, p.state)

	{ // Блокируем для проверки статусов
		p.mx.RLock()

		// Добавление task разрешено только в определенных статусах
		if !(p.state == POOL_STATE_BG_RUNNING || p.state == POOL_STATE_BG_PAUSED) {
			//_log.Info("Pool - has incorrect state to add new task: ExternalId, PoolName, ActiveTaskCount, State", p.externalId, p.name, len(p.taskQueueCh), p.state)
			err = _err.NewTyped(_err.ERR_WORKER_POOL_ADD_TASK_INCORRECT_STATE, p.externalId, p.state, "NEW, RUNNING_BG, PAUSED_BG").PrintfError()
			p.mx.RUnlock()
			return err
		}

		p.mx.RUnlock()
	} // Блокируем для проверки статусов

	// Функция восстановления после паники
	defer func() {
		if r := recover(); r != nil {
			err = _recover.GetRecoverError(r, p.externalId)
		}
	}()

	// Счетчик ожиданий отправки в очередь - увеличить
	_metrics.IncWPAddTaskWaitCountVec(p.name)

	// Очередь имеет ограниченный размер - возможно ожидание, пока не появится свободное место
	p.taskQueueCh <- task

	// Счетчик ожиданий отправки в очередь - отправили - уменьшить
	_metrics.DecWPAddTaskWaitCountVec(p.name)

	return nil
}

// RunOnline запускает задачи в обработку через новые фоновые обработчики
func (p *Pool) RunOnline(externalId uint64, tasks []*Task, shutdownTimeout time.Duration) (err error) {
	if p == nil {
		return _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, externalId, "Nil Pool pointer").PrintfError()
	}

	// Пустую очередь задач игнорируем
	if tasks == nil || len(tasks) == 0 {
		return nil
	}

	{ // блокируем для проверки и установки статусов
		p.mx.Lock()

		// уже запущенный pool запустить повторно нельзя
		if p.state == POOL_STATE_NEW {
			p.setStateUnsafe(POOL_STATE_ONLINE_RUNNING)
			p.externalId = externalId
			p.mx.Unlock()
		} else {
			//_log.Info("Pool online - has incorrect state to run: ExternalId, PoolName, ActiveTaskCount, State", externalId, p.name, len(p.taskQueueCh), p.state)
			err = _err.NewTyped(_err.ERR_WORKER_POOL_RUN_INCORRECT_STATE, p.externalId, p.name, p.state, "NEW").PrintfError()
			p.mx.Unlock()
			return err
		}
	} // блокируем для проверки и установки статусов

	// Работаем в изолированном от родительского контексте
	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.stopCh = make(chan interface{}, 1)
	//defer close(p.stopCh) закрывать канал нужно в том месте, где отправляется сигнал

	// Функция восстановления после глобальной паники и закрытия контекста
	defer func() {
		if r := recover(); r != nil {
			//_log.Info("Pool online - RECOVER FROM PANIC: ExternalId, PoolName, ActiveTaskCount, State", p.externalId, p.name, len(p.taskQueueCh), p.state)
			err = _recover.GetRecoverError(r, p.externalId)
			p.mx.Lock()
			defer p.mx.Unlock()
			p.setStateUnsafe(POOL_STATE_RECOVER_ERR)
			_ = p.shutdownUnsafe(true, shutdownTimeout) // экстренная остановка, ошибку игнорируем
		}

		if p.cancel != nil {
			//_log.Debug("Pool online - STOPPED - close context: ExternalId, PoolName, ActiveTaskCount, State", p.externalId, p.name, len(p.taskQueueCh), p.state)
			p.cancel()
		}
	}()

	p.workers = make(map[int]*Worker, p.workerConcurrency)
	p.workerErrCh = make(chan *WorkerError, p.workerConcurrency) // достаточно по одной ошибке на worker
	p.taskQueueCh = make(chan *Task, len(tasks))
	p.taskQueueSize = len(tasks)

	var wg sync.WaitGroup // все worker стартуют в рамках одной WaitGroup

	//_log.Debug("Pool online - START: ExternalId, PoolName, ActiveTaskCount, State", p.externalId, p.name, len(tasks), p.state)

	// Стартуем worker, передаем им канал ошибок
	for workerCnt := 1; workerCnt <= p.workerConcurrency; workerCnt++ {
		worker := newWorker(p.ctx, p, p.taskQueueCh, uint(workerCnt), externalId, p.workerErrCh, p.workerTimeout)

		p.workers[workerCnt] = worker

		// Увеличиваем счетчик WaitGroup
		wg.Add(1)

		// worker стартуем в фоне в рамках WaitGroup, очередь пока пустая - worker будут ждать задач
		go worker.run(&wg)
	}

	// Наполним очередь задач, worker сразу начнут их выполнение
	for i := range tasks {
		p.taskQueueCh <- tasks[i]
	}

	// Дополнительных задач не ожидаем - закрытие канала - это сигнал для worker об остановке по завершению всех задач из очереди
	close(p.taskQueueCh)

	// В фоне ожидаем закрытия родительского контекста или контекста pool
	if p.parentCtx != nil {
		go func() {
			select {
			case <-p.stopCh:
				//_log.Info("Pool online - STOP - got quit signal: ExternalId, PoolName, ActiveTaskCount, State", p.externalId, p.name, len(p.taskQueueCh), p.state)
			case <-p.parentCtx.Done():
				// Закрылся родительский контекст - останавливаем все worker, должна разблокироваться wg
				//_log.Info("Pool online - STOP - got parent context close: ExternalId, PoolName, ActiveTaskCount, State", p.externalId, p.name, len(p.taskQueueCh), p.state)
				p.mx.Lock()
				// ошибки будут переданы через именованную переменную возврата
				err = p.shutdownUnsafe(true, shutdownTimeout)
				p.mx.Unlock()
			case <-p.ctx.Done():
				// Закрытие контекста pool - нормальная ситуация, если pool отработал штатно, нужно выйти, чтобы не оставляют за собой "подвисших" горутин
				//_log.Debug("Pool online - STOP - got context close: ExternalId, PoolName, ActiveTaskCount, State", p.externalId, p.name, len(p.taskQueueCh), p.state)
			}
		}()
	}

	// Ожидаем выполнения всего объема работ, либо аварийной остановки worker
	wg.Wait()

	{ // блокируем для проверки статусов и завершения
		p.mx.Lock()

		// Если были выполнены не все задачи, то сформируем ошибку
		if len(p.taskQueueCh) != 0 {
			//_log.Info("Pool online - incomplete done of task queue: ExternalId, PoolName, ActiveTaskCount, State", externalId, p.name, len(p.taskQueueCh), p.state)
			p.setStateUnsafe(POOL_STATE_INCOMPLETE_DONE)
			err = _err.NewTyped(_err.ERR_WORKER_POOL_INCOMPLETE_DONE, p.externalId, p.name).PrintfError()
		} else {
			//_log.Debug("Pool online - complete done of task queue: ExternalId, PoolName, ActiveTaskCount, State", externalId, p.name, len(p.taskQueueCh), p.state)
			p.setStateUnsafe(POOL_STATE_ONLINE_DONE)
		}

		p.mx.Unlock()
	} // блокируем для проверки статусов и завершения

	return err
}

// RunBG запускает pool в фоне
func (p *Pool) RunBG(externalId uint64, shutdownTimeout time.Duration) (err error) {
	if p == nil {
		return _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, externalId, "Nil Pool pointer").PrintfError()
	}

	{ // блокируем для проверки и установки статусов
		p.mx.Lock()

		// уже запущенный pool запустить повторно нельзя
		if p.state == POOL_STATE_NEW {
			p.setStateUnsafe(POOL_STATE_BG_RUNNING)
			p.externalId = externalId
			p.mx.Unlock()
		} else {
			//_log.Info("Pool background - has incorrect state to run: ExternalId, PoolName, ActiveTaskCount, State", externalId, p.name, len(p.taskQueueCh), p.state)
			err = _err.NewTyped(_err.ERR_WORKER_POOL_RUN_INCORRECT_STATE, p.externalId, p.name, p.state, "NEW").PrintfError()
			p.mx.Unlock()
			return err
		}
	} // блокируем для проверки и установки статусов

	// Работаем в изолированном от родительского контексте
	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.stopCh = make(chan interface{}, 1)
	//defer close(p.stopCh) закрывать канал нужно в том месте, где отправляется сигнал
	p.workers = make(map[int]*Worker, p.workerConcurrency)
	p.workerErrCh = make(chan *WorkerError, p.workerConcurrency) // достаточно по одной ошибке на worker
	p.taskQueueCh = make(chan *Task, p.taskQueueSize)

	// Функция восстановления после глобальной паники и закрытия контекста
	defer func() {
		if r := recover(); r != nil {
			//_log.Info("Pool background - RECOVER FROM PANIC: ExternalId, PoolName, ActiveTaskCount, State", p.externalId, p.name, len(p.taskQueueCh), p.state)
			err = _recover.GetRecoverError(r, p.externalId)
			p.mx.Lock()
			defer p.mx.Unlock()
			p.setStateUnsafe(POOL_STATE_RECOVER_ERR)
			_ = p.shutdownUnsafe(true, shutdownTimeout) // экстренная остановка, ошибку игнорируем
		}

		// Дополнительных задач не ожидаем - закрытие канала
		close(p.taskQueueCh)

		if p.cancel != nil {
			//_log.Debug("Pool background - STOPPED - close context: ExternalId, PoolName, ActiveTaskCount, State", p.externalId, p.name, len(p.taskQueueCh), p.state)
			p.cancel()
		}
	}()

	//_log.Info("Pool background - START: ExternalId, PoolName, ActiveTaskCount", p.externalId, p.name, len(p.taskQueueCh))

	// Стартуем worker, передаем им канал ошибок
	for workerCount := 1; workerCount <= p.workerConcurrency; workerCount++ {
		worker := newWorker(p.ctx, p, p.taskQueueCh, uint(workerCount), p.externalId, p.workerErrCh, p.workerTimeout)

		p.workers[workerCount] = worker

		go worker.run(nil) // без WaitGroup
	}

	// Ожидаем ошибки от worker, закрытия контекста или остановки pool
	for {
		select {
		case workerErr, ok := <-p.workerErrCh:
			if ok { // канал открыт - нормальная работа pool
				//_log.Info("Pool background - WORKER ERROR - got error from worker restart it: ExternalId, PoolName, ActiveTaskCount", p.externalId, p.name, len(p.taskQueueCh))
				_ = _err.WithCauseTyped(_err.ERR_WORKER_POOL_WORKER_ERROR, p.externalId, workerErr.err, p.name, workerErr.worker.id, workerErr.err.Error()).PrintfError()
				if workerErr.worker != nil {
					go workerErr.worker.run(nil) // стартуем worker заново
				}
			} else { // канал закрыт - нормальная ситуация при остановке pool
				return nil
			}
		case <-p.stopCh:
			//_log.Info("Pool background - STOP - got quit signal: ExternalId, PoolName, ActiveTaskCount, State", p.externalId, p.name, len(p.taskQueueCh), p.state)
			return nil
		case <-p.parentCtx.Done():
			// Закрылся родительский контекст - останавливаем все worker, должна разблокироваться wg
			//_log.Info("Pool background - STOP - got parent context close: ExternalId, PoolName, ActiveTaskCount, State", p.externalId, p.name, len(p.taskQueueCh), p.state)
			p.mx.Lock()
			// ошибки будут переданы через именованную переменную возврата
			err = p.shutdownUnsafe(true, shutdownTimeout)
			p.mx.Unlock()
			return err
		}
	}
}

// Stop закрывает контекст и останавливает workers
func (p *Pool) Stop(hardShutdown bool, shutdownTimeout time.Duration) (err error) {
	if p == nil {
		return _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, _err.ERR_UNDEFINED_ID, "Nil Pool pointer").PrintfError()
	}

	p.mx.Lock()
	defer p.mx.Unlock()

	// Останавливать можно только в определенных статусах
	if p.state != POOL_STATE_SHUTDOWN {
		//_log.Info("Pool online - STOP - got quit signal: ExternalId, PoolName, ActiveTaskCount, State", p.externalId, p.name, len(p.taskQueueCh), p.state)
		err = p.shutdownUnsafe(hardShutdown, shutdownTimeout)

		// Отправляем сигнал и закрываем канал - если pool ни разу не запускался, то p.stopCh будет nil
		if p.stopCh != nil {
			p.stopCh <- true
			close(p.stopCh)
		}

		return err
	}
	return nil
}

// shutdownUnsafe закрывает контекст и останавливает workers
func (p *Pool) shutdownUnsafe(hardShutdown bool, shutdownTimeout time.Duration) (err error) {

	// исключить повторную остановку
	if p.state != POOL_STATE_SHUTDOWN {
		//_log.Debug("Pool - SHUTTING DOWN : ExternalId, PoolName, ActiveTaskCount, State", p.externalId, p.name, len(p.taskQueueCh), p.state)

		p.setStateUnsafe(POOL_STATE_SHUTTING_DOWN)
		// временная метка начала остановки
		//tic := time.Now()

		// Функция восстановления после паники
		defer func() {
			if r := recover(); r != nil {
				err = _recover.GetRecoverError(r, p.externalId)
			}

			p.setStateUnsafe(POOL_STATE_SHUTDOWN)
		}()

		// Запускаем остановку worker и ожидаем shutdownTimeout
		p.stopWorkersUnsafe(hardShutdown, shutdownTimeout)

		{ // Проверим ошибки от worker, которые накопились в канале
			close(p.workerErrCh) // Закрываем канал ошибок worker

			if len(p.workerErrCh) != 0 {
				//_log.Info("Pool - SHUTDOWN - ERROR: ExternalId, PoolName, ErrorCount, duration", p.externalId, p.name, len(p.workerErrCh), time.Now().Sub(tic))

				// накопленные ошибки worker передадим на уровень вверх
				for workerErr := range p.workerErrCh {
					//_log.Debug("Pool online - DONE - Worker error: error", workerErr.err.Error())
					err = _err.WithCauseTyped(_err.ERR_WORKER_POOL_WORKER_ERROR, p.externalId, workerErr.err, p.name, workerErr.worker.id, workerErr.err.Error())
				}
				return err
			} else {
				//_log.Debug("Pool - SHUTDOWN - SUCCESS: ExternalId, PoolName, ActiveTaskCount, duration", p.externalId, p.name, len(p.taskQueueCh), time.Now().Sub(tic))
				return nil
			}
		} // Проверим ошибки от worker, которые накопились в канале
	}
	return nil
}

// stopWorkersUnsafe останавливает запущенных в фоне worker
func (p *Pool) stopWorkersUnsafe(hardShutdown bool, shutdownTimeout time.Duration) {

	//_log.Debug("Pool - STOP workers: ExternalId, PoolName, ActiveTaskCount, State", p.externalId, p.name, len(p.taskQueueCh), p.state)

	// Команда на остановку worker без ожидания завершения очереди, дорабатывает только текущая задача
	// Время выполнения текущей задачи, может превышать отведенный для остановки лимит
	for _, worker := range p.workers {
		worker.stop(hardShutdown)
	}

	stopStartTime := time.Now() // отсчет времени от начала остановки

	// ждем остановки всех workers или предельного shutdownTimeout
	for {

		// Превышено предельное время "жесткой" остановки, в противном случае ждем отработки всех worker
		if shutdownTimeout != 0 && time.Now().After(stopStartTime.Add(shutdownTimeout)) {
			//_log.Info("Pool - STOP WORKER INTERRUPT - exceeded StopTimeout: ExternalId, PoolName, ActiveTaskCount, StopTimeout, State", p.externalId, p.name, len(p.taskQueueCh), shutdownTimeout, p.state)
			p.setStateUnsafe(POOL_STATE_TERMINATE_TIMEOUT)
			return
		}

		anyNonStoppedWorker := false // есть ли не остановленные worker

		// Проверим все worker, кто еще не остановился
		for _, worker := range p.workers {
			if worker.state != WORKER_STATE_TERMINATED {
				//_log.Debug("Pool - WORKER STILL WORKING: ExternalId, PoolName, WorkerId, ActiveTaskCount, State", p.externalId, p.name, worker.id, len(p.taskQueueCh), p.state)
				anyNonStoppedWorker = true // Есть хоть один не остановленный
				break
			}
		}

		// Все worker остановлены
		if !anyNonStoppedWorker {
			//_log.Info("Pool - ALL WORKER STOPPED: ExternalId, PoolName, ActiveTaskCount, State", p.externalId, p.name, len(p.taskQueueCh), p.state)
			return
		}

		// Задержка перед повторной проверкой
		time.Sleep(time.Millisecond + 10)
	}
}
