package httphandler

import (
	"context"
	"net/http"
	"strconv"

    _ctx "github.com/romapres2010/goapp/pkg/common/ctx"
    _http "github.com/romapres2010/goapp/pkg/common/httpservice"
    _log "github.com/romapres2010/goapp/pkg/common/logger"
)

// ReadyHandler handle to test readinessProbe
func (s *Service) ReadyHandler(w http.ResponseWriter, r *http.Request) {
	_log.Debug("START   ==================================================================================")

	// Запускаем обработчик, возврат ошибки игнорируем
	_ = s.httpService.Process(true, "GET", w, r, func(ctx context.Context, requestBuf []byte, buf []byte) ([]byte, _http.Header, int, error) {
		requestID := _ctx.FromContextHTTPRequestID(ctx) // RequestID передается через context

		_log.Debug("START: requestID", requestID)

		// формируем ответ
		header := _http.Header{}
		header[_http.HEADER_CUSTOM_ERR_CODE] = _http.HEADER_CUSTOM_ERR_CODE_SUCCESS
		header[_http.HEADER_CUSTOM_REQUEST_ID] = strconv.FormatUint(requestID, 10)

		// Считаем параметры из заголовка сообщения и перенесем их в ответный заголовок
		for key := range r.Header {
			header[key] = r.Header.Get(key)
		}

		_log.Debug("SUCCESS", requestID)

		// входной буфер возвращаем в качестве выходного
		return requestBuf, header, http.StatusOK, nil
	})

	_log.Debug("SUCCESS ==================================================================================")
}
