package httphandler

import (
    "context"
    "testing"
    "time"

    _log "github.com/romapres2010/goapp/pkg/common/logger"
    _wp "github.com/romapres2010/goapp/pkg/common/workerpool"
    _wpservice "github.com/romapres2010/goapp/pkg/common/workerpoolservice"
)

func BenchmarkCalculateFactorial(b *testing.B) {

    var wpServiceCfg = &_wpservice.Config{
        TotalTimeout:    100 * time.Millisecond,
        ShutdownTimeout: 30 * time.Second,
        WPCfg: _wp.Config{
            TaskQueueSize:     0,
            TaskTimeout:       20 * time.Second,
            WorkerConcurrency: 8,
            WorkerTimeout:     30 * time.Second,
        },
    }

    var wpService *_wpservice.Service        // сервис worker pool
    var wpServiceErrCh = make(chan error, 1) // канал ошибок для сервиса worker pool
    var err error
    var parentCtx = context.Background()

    // создаем сервис обработчиков
    if wpService, err = _wpservice.New(parentCtx, "WorkerPool - background", wpServiceErrCh, wpServiceCfg); err != nil {
        return
    }

    // запускаем сервис обработчиков - паники должны быть обработаны внутри
    go func() { wpServiceErrCh <- wpService.Run() }()
    time.Sleep(10 * time.Millisecond)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        //b.StopTimer()
        wpFactorialReqResp := &WpFactorialReqResp{
            // NumArray: []int{50, 50, 50, 50, 50, 50, 50, 50, 50, 50, 50, 50, 50, 50},
            NumArray: []uint64{3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3},
        }
        //b.StartTimer()
        _ = calculateFactorial(parentCtx, wpService, 0, wpFactorialReqResp, "bg")
    }
    b.StopTimer()

    //Останавливаем обработчик worker pool
    if err = wpService.Shutdown(true, wpServiceCfg.ShutdownTimeout); err != nil {
        _log.ErrorAsInfo(err) // дополнительно логируем результат остановки
    }
}
