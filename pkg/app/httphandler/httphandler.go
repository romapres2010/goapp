package httphandler

import (
    "context"

    _ctx "github.com/romapres2010/goapp/pkg/common/ctx"
    _err "github.com/romapres2010/goapp/pkg/common/error"
    _http "github.com/romapres2010/goapp/pkg/common/httpservice"
    _log "github.com/romapres2010/goapp/pkg/common/logger"
)

// Service represent HTTP service
type Service struct {
    ctx    context.Context    // корневой контекст при инициации сервиса
    cancel context.CancelFunc // функция закрытия глобального контекста
    cfg    *Config            // конфигурационные параметры

    // вложенные сервисы
    httpService *_http.Service // сервис HTTP
}

// Config represent HTTP Service configurations
type Config struct {
}

// New create new HTTP service
func New(ctx context.Context, cfg *Config,
    httpService *_http.Service) (*Service, error) {

    requestID := _ctx.FromContextHTTPRequestID(ctx) // RequestID передается через context

    _log.Info("Creating new HTTP service")

    { // входные проверки
        if cfg == nil {
            return nil, _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, requestID, "if cfg == nil {}").PrintfError()
        }
        if httpService == nil {
            return nil, _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, requestID, "if httpService == nil {}").PrintfError()
        }
    } // входные проверки

    service := &Service{
        cfg:         cfg,
        httpService: httpService,
    }

    // создаем контекст с отменой
    if ctx == nil {
        service.ctx, service.cancel = context.WithCancel(context.Background())
    } else {
        service.ctx, service.cancel = context.WithCancel(ctx)
    }

    _log.Info("HTTP service was created")
    return service, nil
}
