package httpservice

import (
	"context"
	"net/http"
	"strconv"

    _ctx "github.com/romapres2010/goapp/pkg/common/ctx"
    _log "github.com/romapres2010/goapp/pkg/common/logger"
)

// EchoHandler handle echo page with request header and body
func (s *Service) EchoHandler(w http.ResponseWriter, r *http.Request) {
	_log.Debug("START   ==================================================================================")

	// Запускаем обработчик, возврат ошибки игнорируем
	_ = s.Process(true, "POST", w, r, func(ctx context.Context, requestBuf []byte, buf []byte) ([]byte, Header, int, error) {
		requestID := _ctx.FromContextHTTPRequestID(ctx) // RequestID передается через context

		_log.Debug("START: requestID", requestID)

		// формируем ответ
		header := Header{}
		header[HEADER_CUSTOM_ERR_CODE] = HEADER_CUSTOM_ERR_CODE_SUCCESS
		header[HEADER_CUSTOM_REQUEST_ID] = strconv.FormatUint(requestID, 10)

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
