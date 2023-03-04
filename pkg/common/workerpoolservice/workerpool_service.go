package workerpoolservice

import (
	"context"
	"sync"
	"time"

	_err "github.com/romapres2010/goapp/pkg/common/error"
	_recover "github.com/romapres2010/goapp/pkg/common/recover"
	_wp "github.com/romapres2010/goapp/pkg/common/workerpool"
)

// Service represent WorkerPool service
type Service struct {
	ctx    context.Context    // корневой контекст при инициации сервиса
	cancel context.CancelFunc // функция закрытия глобального контекста
	errCh  chan<- error       // канал ошибок
	stopCh chan struct{}      // канал подтверждения об успешном закрытии сервиса
	name   string             // наименование WorkerPool

	cfg  *Config // конфигурационные параметры
	pool *_wp.Pool
}

// Config конфигурационные настройки
type Config struct {
	TotalTimeout    time.Duration `yaml:"total_timeout" json:"total_timeout"`       // максимальное время обработки
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" json:"shutdown_timeout"` // максимальное время жестокой остановки worker
	WPCfg           _wp.Config    `yaml:"worker_pool" json:"worker_pool"`           // конфигурационные параметры
}

// New - create WorkerPool service
func New(parentCtx context.Context, name string, errCh chan<- error, cfg *Config) (*Service, error) {
	var externalId = _err.ERR_UNDEFINED_ID
	var err error

	//_log.Info("Creating new WorkerPool service: WorkerPoolName", name)

	{ // входные проверки
		if cfg == nil {
			return nil, _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, externalId, "if cfg == nil {}").PrintfError()
		}
	} // входные проверки

	// Создаем новый сервис
	service := &Service{
		cfg:    cfg,
		errCh:  errCh,
		name:   name,
		stopCh: make(chan struct{}, 1), // канал подтверждения об успешном закрытии сервиса
	}

	// создаем контекст с отменой
	if parentCtx == nil {
		service.ctx, service.cancel = context.WithCancel(context.Background())
	} else {
		service.ctx, service.cancel = context.WithCancel(parentCtx)
	}

	//_log.Info("Create worker pool: WorkerPoolName, WorkerConcurrency", service.name, service.cfg.WPCfg.WorkerConcurrency)

	service.pool, err = _wp.NewPool(parentCtx, externalId, service.name, &service.cfg.WPCfg)
	if err != nil {
		return nil, err
	}

	//_log.Info("WorkerPool service was created: WorkerPoolName", service.name)

	return service, nil
}

// AddTask добавляет таски в pool - если очередь переполнена, то ожидание
func (s *Service) AddTask(task *_wp.Task) error {
	if task != nil {
		err := s.pool.AddTask(task)
		if err != nil {
			return _err.WithCauseTyped(_err.ERR_ERROR, task.GetExternalId(), err, "UTCE Calculator was shutting down")
		}
		return nil
	}
	return _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, _err.ERR_UNDEFINED_ID, "if task != nil {}")
}

// GetWPConfig конфигурация workerpool
func (s *Service) GetWPConfig() *_wp.Config {
	return &s.cfg.WPCfg
}

// RunTasksGroupWG - запустить группу задач в отдельной WG
func (s *Service) RunTasksGroupWG(externalId uint64, tasks []*_wp.Task, taskGroupName string) (err error) {

	// Функция восстановления после паники
	defer func() {
		if r := recover(); r != nil {
			err = _recover.GetRecoverError(r, externalId, s.name)
		}
	}()

	//_log.Debug("Pool service - START: ExternalId, WorkerPoolName, TaskName", externalId, s.name, taskGroupName)

	var wgCnt int                                   // сколько task было отправлено = wg.Add
	var wg sync.WaitGroup                           // все task выполняются в одной WaitGroup
	var doneCh = make(chan interface{}, len(tasks)) // канал ответов об окончании задач
	defer close(doneCh)

	var startTime = time.Now() // отсчет времени от начала обработки

	// добавляем задачи в определенную группу, группа задач определяется каналом завершения
	for _, task := range tasks {
		if task != nil { // пустые task игнорируем

			// Превышено максимальное время выполнения
			if s.cfg.TotalTimeout > 0 && time.Now().After(startTime.Add(s.cfg.TotalTimeout)) {
				err = _err.NewTyped(_err.ERR_WORKER_POOL_TIMEOUT_ERROR, externalId, task.GetExternalId(), s.cfg.TotalTimeout).PrintfError()
				return err
			}

			task.SetDoneCh(doneCh) // установим общий канал окончания

			// Отправляем в канал очереди задач, если канал заполнен - то ожидание
			if err = s.AddTask(task); err != nil {
				// Возможна ситуация, когда pool уже остановлен и закрыт канал очереди задач
				return err
			}
			wgCnt++
			wg.Add(1)
		}
	}

	// Управление wg.Done() не переносится в задачу - выполняется в отдельной горутине
	go func() {
		// количество завершенных задач
		var doneCnt int
		var timer *time.Timer
		if s.cfg.TotalTimeout > 0 {
			timer = time.NewTimer(s.cfg.TotalTimeout)
		} else {
			timer = time.NewTimer(_wp.POOL_MAX_TIMEOUT) // максимальное время ожидания
		}
		defer timer.Stop()

		for {
			// Ожидаем завершения обработки всех задач в WaitGroup или TotalTimeout
			select {
			case _, ok := <-doneCh:
				if ok { // канал открыт
					if doneCnt < wgCnt {
						wg.Done()
						doneCnt++
					} else { // Избежать отрицательного wg.Done() - ошибка во внутренней логике - для анализа
						doneCnt++
						err = _err.NewTyped(_err.ERR_WORKER_POOL_TASK_COUNT_ERROR, externalId, taskGroupName, wgCnt, doneCnt).PrintfError()
						return
					}
				} else {
					return // канал закрыт
				}
			case <-s.ctx.Done():
				//_log.Info("Worker pool service -  got context close: externalId", externalId)
				err = _err.NewTyped(_err.ERR_WORKER_POOL_CONTEXT_CLOSED, externalId, "Worker pool service -  got context close")
				// Уменьшить счетчик, чтобы разблокировать родительскую горутину
				for doneCnt < wgCnt {
					wg.Done()
					doneCnt++
				}
			case <-timer.C:
				err = _err.NewTyped(_err.ERR_WORKER_POOL_TIMEOUT_ERROR, externalId, externalId, s.cfg.TotalTimeout).PrintfError()
				// Уменьшить счетчик, чтобы разблокировать родительскую горутину
				for doneCnt < wgCnt {
					wg.Done()
					doneCnt++
				}
				// !!! вариант с time.After(s.cfg.TotalTimeout) использовать нельзя, так как будут оставаться "повешенными" таймеры
				//case <-time.After(s.cfg.TotalTimeout):
				//    err = _err.NewTyped(_err.ERR_WORKER_POOL_TIMEOUT_ERROR, externalId, externalId, s.cfg.TotalTimeout).PrintfError()
				//    // Уменьшить счетчик, чтобы разблокировать родительскую горутину
				//    for doneCnt < wgCnt {
				//        wg.Done()
				//        doneCnt++
				//    }
				// !!! вариант с time.After(s.cfg.TotalTimeout) использовать нельзя, так как будут оставаться "повешенными" таймеры
			}
		}
	}()

	// Ожидать выполнения всех задач task в группе
	wg.Wait()

	if err != nil {
		//_log.Debug("Pool service - ERROR: externalId, WorkerPoolName, TaskName, error", externalId, s.name, taskGroupName, err.Error())
	} else {
		//_log.Debug("Pool service - SUCCESS: externalId, WorkerPoolName, TaskName", externalId, s.name, taskGroupName)
	}

	return err
}

// Run  - wait for error or exit
func (s *Service) Run() (err error) {
	// Функция восстановления после паники
	defer func() {
		r := recover()
		if r != nil {
			err = _recover.GetRecoverError(r, _err.ERR_UNDEFINED_ID, s.name)
		}
	}()
	//_log.Info("Run WorkerPool service: WorkerPoolName", s.name)

	return s.pool.RunBG(0, s.cfg.ShutdownTimeout)
}

// Shutdown shutting down service
func (s *Service) Shutdown(hardShutdown bool, shutdownTimeout time.Duration) (err error) {
	//_log.Info("Shutdown WorkerPool service: WorkerPoolName", s.name)

	defer s.cancel()

	{ // закрываем вложенные сервисы
		if shutdownTimeout > s.cfg.ShutdownTimeout {
			shutdownTimeout = s.cfg.ShutdownTimeout
		}
		var poolShutdownMode _wp.PoolShutdownMode
		if hardShutdown {
			poolShutdownMode = _wp.POOL_SHUTDOWN_HARD
		} else {
			poolShutdownMode = _wp.POOL_SHUTDOWN_SOFT
		}
		err = s.pool.Stop(poolShutdownMode, shutdownTimeout) // ожидание остановки worker
	} // закрываем вложенные сервисы

	if err != nil {
		//_log.Info("WorkerPool service shutdown error: WorkerPoolName, error:", s.name, err.Error())
	} else {
		//_log.Info("WorkerPool service shutdown successfully: WorkerPoolName", s.name)
	}

	s.pool.PrintTaskPoolStats()

	return err
}
