[# Шаблон backend сервера на Golang — часть 5 - Worker pool](https://habr.com/ru/post/720286/)

[Третья часть](https://habr.com/ru/post/716634/) посвящена развертыванию шаблона в Docker, Docker Compose, Kubernetes (kustomize).

[Пятая часть](https://habr.com/ru/post/720286/) посвящена простому Worker pool.

Представленный в шаблоне Worker pool дополнительно реализует следующую функциональность:
- Возможность экстренной остановки Worker pool в целом 
  - явно по команде остановки
  - автоматически по закрытию корневого контекста
  - контроль timeout остановки работающих worker
- Возможность экстренной остановки отдельного worker
  - явно по команде остановки (жесткий режим или с ожиданием завершения текущей task)
  - автоматически по закрытию корневого контекста
- Возможность экстренной остановки отдельного task
  - явно по команде остановки
  - по предельному таймауту
- Автоматический перезапуск сбойного worker
- Сбор метрик [prometheus](https://prometheus.io/) по загрузке worker, очереди задач и производительности выполнения по типам task
- Возможность управления группами task в рамках единого Worker pool
- Работа с двумя типами task
  - "Короткие" - не контролируется предельный timeout выполнения и их нельзя прервать 
  - "Длинные" - контролируется предельный timeout выполнения и их можно прервать

Представленный Worker pool был существенно оптимизирован по накладным расходам:
- Для "коротких" task - 300-500 ns/op, 35 B/op, 0 allocs/op
- Для "длинных" task - 600-900 ns/op, 60 B/op, 1 allocs/op

Для task, которые должны выполняться быстрее 100 ns/op представленный Worker pool использовать не эффективно. 

Ссылка на [репозиторий проекта](https://github.com/romapres2010/goapp).

Шаблон goapp в репозитории полностью готов к развертыванию в Docker, Docker Compose, Kubernetes (kustomize), Kubernetes (helm).

<cut />

## Содержание
1. Архитектура Worker pool
2. Подходы к остановке приложения (микросервиса)
3. Структура Task
4. Настройка Worker pool через конфиг
5. Оптимизация накладных расходов Worker pool
6. Пример использования Worker pool
7. Нагрузочное тестирование Worker pool
 
<cut />

## . Архитектура Worker pool
В основе лежит концепция из статьи [Ahad Hasan](https://hackernoon.com/concurrency-in-golang-and-workerpool-part-2-l3w31q7). 

При развертывании приложения в Kubernetes столкнулись с особенностями.
- при росте нагрузки Horizontal Autoscaler (HA) может создавать новые Pod c приложением и перенаправлять на него часть запросов.
- при снижении нагрузки (по памяти или загрузке процессора), Horizontal Autoscaler останавливает Pod c приложением.

В нашем приложении Worker pool, использовался для двух типов задач:
- высоконагруженных расчетов с периодической нагрузкой
- длительных (1-30 сек) и слабонагруженных задач взаимодействия с внешними сервисами

В периоды высокой нагрузки, Horizontal Autoscaler создавал 2-5 новых Pod, а через 30-60 минут удалял ненужные. Pod останавливаются произвольным образом, в результате мы получали обрывы соединений и отказ в обслуживании для длительных операции.

Правильный вариант решения такой проблемы - это разнесение разных типов задач на разные микросервисы. Но вместе с этим, пришлось серьезно перепроектировать Worker pool.


## . Подходы к остановке приложения (микросервиса) 
Условно, можно выделить следующие подходы к остановке приложения:
- "Light" - все начатые к обработке и все взятые в очередь задачи должны быть завершены, новые задачи не принимаются. Потребители по новым запросам получают отказ в обслуживании.
- "Soft" - только начатые к обработке задачи должны быть завершены, новые задачи не принимаются, оставшиеся в очереди задачи останавливаются с ошибкой. Потребители по новым запросам и запросам не начатым обрабатываться получают отказ в обслуживании.
- "Soft + timeout" - сначала отрабатывает "Soft", если не уложились в timeout, то срабатывает "Hard".   
- "Hard" - экстренно прерывается обработка всех задач, как начатых, так и находящихся в очереди. Потребители получают отказ в обслуживании. 
- "Crash" - приложение удаляется KILL -9. Сетевые соединения разрываются. Потребители не получают дополнительной информации кроме разрыва соединения.

Если приложение stateless, то, желательно использовать подход "Crash" или "Hard". Потребители всегда смогут отправить повторные запросы и их обработает другой Pod.

Если приложение stateful, и завершить начатые задачи в режиме "soft" невозможно, то нужно сделать пометку о необходимости компенсационного действия. Компенсационные действия может выполнять само приложение при повторном запуске, либо отдельный служебный сервис.  

Шаблон Worker pool в репозитории поддерживается варианты остановки "Light", "Soft", "Soft + timeout", "Hard". по умолчанию настроен режим "Soft + timeout". 

## . Структура Task

[Task](https://github.com/romapres2010/goapp/blob/master/pkg/common/workerpool/task.go) - содержит входные параметры задачи, функцию обработчик, результаты выполнения, каналы для управления и таймер для контроля timeout.

Основные задачи Task:
- Запустить функцию-обработчик и передать ей входные данные
- Контролировать результат выполнения функции-обработчика
- Информировать "внешний мир" о завершении выполнения функции-обработчика
- Контролировать время выполнения функции-обработчика по timeout, при необходимости прервать выполнение
- Перехватить panic от функции-обработчика и обработать ошибку
- Контролировать команду на остановку со стороны Worker pool
- Информировать функцию-обработчика о необходимости срочной остановки

``` go
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

	f func(context.Context, context.Context, ...interface{}) (error, []interface{}) // функция обработчик task

	mx sync.RWMutex
}
}
```

Task управляется следующей статусной моделью.
``` go
type TaskState int

const (
	TASK_STATE_NEW                    TaskState = iota // task создан
	TASK_STATE_IN_PROCESS                              // task выполняется
	TASK_STATE_DONE_SUCCESS                            // task завершился
	TASK_STATE_RECOVER_ERR                             // task остановился из-за паники
	TASK_STATE_TERMINATED_STOP_SIGNAL                  // task остановлен по причине получения сигнала об остановке
	TASK_STATE_TERMINATED_CTX_CLOSED                   // task остановлен по причине закрытия контекста
	TASK_STATE_TERMINATED_TIMEOUT                      // task остановлен по причине превышения timeout
)
```

Task запускается из goroutine worker
- Заблокируем task на время запуска, чтобы исключить одновременное использование одного указателя task
- Проверим, что запускать можно только новый task
- Cтартуем defer функцию для обработки паники task и информирования "внешнего мира" о завершении работы task в отдельный канал
- "Короткие" task (timeout < 0) не контролируем по timeout. Их нельзя прервать. Функция-обработчик запускается в goroutine worker
- "Длинные" task (timeout >= 0) запускаем в фоне и ожидаем завершения в отдельный локальный канал. Функция-обработчик получает родительский контекст и локальный контекст task. Локальный контекст task нужно контролировать в обработчике для определения необходимости остановки. Ожидаем завершения функции обработчика, наступления timeout или команды на закрытие task

``` go
func (ts *Task) process(workerID uint, workerTimeout time.Duration) {
	if ts == nil || ts.f == nil {
		return
	}

	// Заблокируем task на время запуска, чтобы исключить одновременное использование одного указателя
	if ts.mx.TryLock() { // Использование TryLock не рекомендуется, но в данном случае это очень удобно
		defer ts.mx.Unlock()
	} else {
		ts.err = _err.NewTyped(_err.ERR_WORKER_POOL_TASK_ALREADY_LOCKED, ts.externalId, ts.name, ts.state).PrintfError()
		return
	}

	// Проверим, что запускать можно только новый task
	if ts.state == TASK_STATE_NEW {
		ts.setStateUnsafe(TASK_STATE_IN_PROCESS)
	} else {
		ts.err = _err.NewTyped(_err.ERR_WORKER_POOL_TASK_INCORRECT_STATE, ts.externalId, ts.name, ts.state, "NEW").PrintfError()
		return
	}

	var tic = time.Now() // временная метка начала обработки task

	// Обрабатываем панику task и информируем "внешний мир" о завершении работы task в отдельный канал
	defer func() {
		if r := recover(); r != nil {
			ts.err = _recover.GetRecoverError(r, ts.externalId, ts.name)
			ts.setStateUnsafe(TASK_STATE_RECOVER_ERR)
		}

		if ts.doneCh != nil {
			ts.doneCh <- struct{}{}
		}
	}()

	if ts.timeout < 0 {
		// "Короткие" task (timeout < 0) не контролируем по timeout. Их нельзя прервать. Функция-обработчик запускается в goroutine worker
		ts.err, ts.responses = ts.f(ts.parentCtx, nil, ts.requests...)
		ts.duration = time.Now().Sub(tic)
		ts.setStateUnsafe(TASK_STATE_DONE_SUCCESS)
		return
	} else {
		// "Длинные" task запускаем в фоне и ожидаем завершения в отдельный локальный канал. Контролируем timeout

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
			ts.duration = time.Now().Sub(tic)
			ts.setStateUnsafe(TASK_STATE_DONE_SUCCESS)
			ts.timer.Stop() // остановим таймер, сбрасывать канал не требуется, так как он не сработал
			return
		case _, ok := <-ts.stopCh:
			if ok {
				// канал был открыт и получили команду на остановку
				ts.err = _err.NewTyped(_err.ERR_WORKER_POOL_STOP_SIGNAL, ts.externalId, fmt.Sprintf("[WorkerId='%v', TaskExternalId='%v', TaskName='%v', WorkerTimeout='%v']", workerID, ts.externalId, ts.name, workerTimeout))
				ts.setStateUnsafe(TASK_STATE_TERMINATED_STOP_SIGNAL)
			} else {
				_log.Error("Task - INTERRUPT - stop chanel closed: WorkerId, TaskExternalId, TaskName, WorkerTimeout", workerID, ts.externalId, ts.name, workerTimeout)
			}
			// Закрываем локальный контекст task - функция обработчика должна корректно отработать это состояние и выполнить компенсационные воздействия
			if ts.cancel != nil {
				ts.cancel()
			}
			return
		case <-ts.timer.C:
			ts.err = _err.NewTyped(_err.ERR_WORKER_POOL_TIMEOUT_ERROR, ts.externalId, ts.id, timeout).PrintfError()
			ts.setStateUnsafe(TASK_STATE_TERMINATED_TIMEOUT)
			// Закрываем локальный контекст task - функция обработчика должна корректно отработать это состояние и выполнить компенсационные воздействия
			if ts.cancel != nil {
				ts.cancel()
			}
			return
		}
	}
}
```