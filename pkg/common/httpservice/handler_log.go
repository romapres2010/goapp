package httpservice

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

    _ctx "github.com/romapres2010/goapp/pkg/common/ctx"
    _err "github.com/romapres2010/goapp/pkg/common/error"
    _log "github.com/romapres2010/goapp/pkg/common/logger"
)

// HTTPLogHandler handle HTTP log mode
func (s *Service) HTTPLogHandler(w http.ResponseWriter, r *http.Request) {
	_log.Debug("START   ==================================================================================")

	// Запускаем обработчик, возврат ошибки игнорируем
	_ = s.Process(false, "POST", w, r, func(ctx context.Context, requestBuf []byte, buf []byte) ([]byte, Header, int, error) {
		requestID := _ctx.FromContextHTTPRequestID(ctx) // RequestID передается через context

		_log.Debug("START: requestID", requestID)

		// считаем текущий HTTP Loger config и поменяем его по ссылке !!!!
		logCurCfg := s.httpLogger.GetConfig()
		if logCurCfg != nil {
			logCfg := *logCurCfg // делаем копию

			{ // обрабатываем HTTP-log из заголовка
				HTTPLogStr := r.Header.Get(HEADER_LOG_HTTP_TRAFFIC_STATUS)
				if HTTPLogStr != "" {
					switch strings.ToUpper(HTTPLogStr) {
					case "TRUE":
						logCfg.Enable = true
					case "FALSE":
						logCfg.Enable = false
					default:
						myerr := _err.NewTyped(_err.ERR_HTTP_HEADER_INCORRECT_BOOL, requestID, HTTPLogStr, HEADER_LOG_HTTP_TRAFFIC_STATUS).PrintfError()
						return nil, nil, http.StatusBadRequest, myerr
					}
				} else {
					logCfg.Enable = false
				}
				_log.Info("Set '"+HEADER_LOG_HTTP_TRAFFIC_STATUS+"'='", HTTPLogStr+"'")
			} // обрабатываем HTTP-log из заголовка

			{ // обрабатываем HTTP-log-type из заголовка
				HTTPLogTypeStr := r.Header.Get(HEADER_LOG_HTTP_TRAFFIC_TYPE)
				// логировать входящие запросы
				if strings.Index(HTTPLogTypeStr, "INREQ") >= 0 {
					logCfg.LogInReq = true
				}
				// логировать исходящие запросы
				if strings.Index(HTTPLogTypeStr, "OUTREQ") >= 0 {
					logCfg.LogOutReq = true
				}
				// логировать входящие ответы
				if strings.Index(HTTPLogTypeStr, "INRESP") >= 0 {
					logCfg.LogInResp = true
				}
				// логировать исходящие ответы
				if strings.Index(HTTPLogTypeStr, "OUTRESP") >= 0 {
					logCfg.LogOutResp = true
				}
				// логировать тело запроса
				if strings.Index(HTTPLogTypeStr, "BODY") >= 0 {
					logCfg.LogBody = true
				}
				_log.Info("Set '"+HEADER_LOG_HTTP_TRAFFIC_TYPE+"'='", HTTPLogTypeStr+"'")
			} // обрабатываем HTTP-log-type из заголовка

			// формируем ответ
			header := Header{}
			header[HEADER_CUSTOM_ERR_CODE] = HEADER_CUSTOM_ERR_CODE_SUCCESS
			header[HEADER_CUSTOM_REQUEST_ID] = strconv.FormatUint(requestID, 10)

			// устанавливаем новый конфиг для HTTP logger
			if err := s.httpLogger.SetConfig(&logCfg); err != nil {
				return nil, nil, http.StatusInternalServerError, err
			} else {
				_log.Debug("SUCCESS", requestID)
				return requestBuf, header, http.StatusOK, nil
			}
		}
		return nil, nil, http.StatusInternalServerError, _err.NewTyped(_err.ERR_COMMON_ERROR, requestID, "Empty current HTTP logger config")
	})

	_log.Debug("SUCCESS ==================================================================================")
}

// HTTPErrorLogHandler handle loging error into HTTP response
func (s *Service) HTTPErrorLogHandler(w http.ResponseWriter, r *http.Request) {
	_log.Debug("START   ==================================================================================")

	// Запускаем обработчик, возврат ошибки игнорируем
	_ = s.Process(false, "POST", w, r, func(ctx context.Context, requestBuf []byte, buf []byte) ([]byte, Header, int, error) {
		requestID := _ctx.FromContextHTTPRequestID(ctx) // RequestID передается через context

		_log.Debug("START: requestID", requestID)

		{ // обрабатываем HTTP-Err-Log из заголовка
			HTTPErrLogStr := r.Header.Get(HEADER_LOG_ERROR_TO_HTTP_TYPE)
			// логировать ошибку в заголовок ответа
			if strings.Index(HTTPErrLogStr, "HEADER") >= 0 {
				s.cfg.HTTPErrorLogHeader = true
			} else {
				s.cfg.HTTPErrorLogHeader = false
			}
			// логировать ошибку в тело ответа
			if strings.Index(HTTPErrLogStr, "BODY") >= 0 {
				s.cfg.HTTPErrorLogBody = true
			} else {
				s.cfg.HTTPErrorLogBody = false
			}
		} // обрабатываем HTTP-Err-Log из заголовка

		_log.Info("Set HTTP Error: HTTPErrorLogHeader, HTTPErrorLogBody", s.cfg.HTTPErrorLogHeader, s.cfg.HTTPErrorLogBody)

		// формируем ответ
		header := Header{}
		header[HEADER_CUSTOM_ERR_CODE] = HEADER_CUSTOM_ERR_CODE_SUCCESS
		header[HEADER_CUSTOM_REQUEST_ID] = strconv.FormatUint(requestID, 10)

		_log.Debug("SUCCESS", requestID)
		return requestBuf, header, http.StatusOK, nil
	})

	_log.Debug("SUCCESS ==================================================================================")
}

// LogLevelHandler handle logging filter
func (s *Service) LogLevelHandler(w http.ResponseWriter, r *http.Request) {
	_log.Debug("START   ==================================================================================")

	// Запускаем обработчик, возврат ошибки игнорируем
	_ = s.Process(false, "POST", w, r, func(ctx context.Context, requestBuf []byte, buf []byte) ([]byte, Header, int, error) {
		requestID := _ctx.FromContextHTTPRequestID(ctx) // RequestID передается через context

		_log.Debug("START: requestID", requestID)

		LogLevelStr := r.Header.Get(HEADER_LOG_LEVEL_GLOBALLOGFILTER)
		switch LogLevelStr {
		case "DEBUG", "ERROR", "INFO":
			_log.SetStdFilterLevel(LogLevelStr)
		default:
			myerr := _err.NewTyped(_err.ERR_INCORRECT_LOG_LEVEL, requestID, LogLevelStr).PrintfError()
			_log.Error(fmt.Sprintf("%+v", myerr))
			return nil, nil, http.StatusBadRequest, myerr
		}

		_log.Info("Set log level", s.cfg.HTTPErrorLogHeader, s.cfg.HTTPErrorLogBody)

		// формируем ответ
		header := Header{}
		header[HEADER_CUSTOM_ERR_CODE] = HEADER_CUSTOM_ERR_CODE_SUCCESS
		header[HEADER_CUSTOM_REQUEST_ID] = strconv.FormatUint(requestID, 10)

		_log.Debug("SUCCESS", requestID)
		return requestBuf, header, http.StatusOK, nil
	})

	_log.Debug("SUCCESS ==================================================================================")
}
