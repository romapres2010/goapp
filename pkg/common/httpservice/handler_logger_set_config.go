package httpservice

import (
	"context"
	"encoding/json"
	"gopkg.in/yaml.v3"
	"net/http"
	"strconv"

	_ctx "github.com/romapres2010/goapp/pkg/common/ctx"
	_err "github.com/romapres2010/goapp/pkg/common/error"
	_log "github.com/romapres2010/goapp/pkg/common/logger"
)

// LoggerSetConfigHandler Сервис отвечает за установку в YAML формате конфигурационных настроек logger
func (s *Service) LoggerSetConfigHandler(w http.ResponseWriter, r *http.Request) {
	_log.Debug("START   ==================================================================================")

	// Запускаем типовой Process, возврат ошибки игнорируем
	_ = s.Process(false, "POST", w, r, func(ctx context.Context, requestBuf []byte, buf []byte) ([]byte, Header, int, error) {
		var requestID uint64 = _ctx.FromContextHTTPRequestID(ctx) // RequestID передается через context
		var id int
		var err error
		var responseBuf []byte
		var loggerCfg _log.Config

		_log.Debug("START: requestID", requestID)

		// Считаем параметры из URL query
		fmt := r.URL.Query().Get("fmt")

		if fmt == "json" {
			if err = json.Unmarshal(requestBuf, &loggerCfg); err != nil {
				err = _err.WithCauseTyped(_err.ERR_JSON_UNMARSHAL_ERROR, requestID, err).PrintfError()
				return nil, nil, http.StatusBadRequest, err
			}
			if err = _log.SetGlobalConfig(loggerCfg); err != nil {
				return nil, nil, http.StatusBadRequest, err
			}
			if responseBuf, err = json.MarshalIndent(_log.GlobalConfig(), "", "    "); err != nil {
				err = _err.WithCauseTyped(_err.ERR_JSON_MARSHAL_ERROR, requestID, err).PrintfError()
				return nil, nil, http.StatusBadRequest, err
			}
		} else if fmt == "yaml" {
			if err = yaml.Unmarshal(requestBuf, &loggerCfg); err != nil {
				err = _err.WithCauseTyped(_err.ERR_XML_UNMARSHAL_ERROR, requestID, err).PrintfError()
				return nil, nil, http.StatusBadRequest, err
			}
			if err = _log.SetGlobalConfig(loggerCfg); err != nil {
				return nil, nil, http.StatusBadRequest, err
			}
			if responseBuf, err = yaml.Marshal(_log.GlobalConfig()); err != nil {
				err = _err.WithCauseTyped(_err.ERR_XML_MARSHAL_ERROR, requestID, err).PrintfError()
				return nil, nil, http.StatusBadRequest, err
			}
		} else {
			return nil, nil, http.StatusInternalServerError, _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, _err.ERR_UNDEFINED_ID, "Allowed only fmt = 'json', 'yaml'", fmt).PrintfError()
		}

		// формируем ответ
		header := Header{}
		header[HEADER_CONTENT_TYPE] = HEADER_CONTENT_TYPE_JSON_UTF8
		header[HEADER_CUSTOM_ERR_CODE] = HEADER_CUSTOM_ERR_CODE_SUCCESS
		header[HEADER_CUSTOM_ID] = strconv.Itoa(id)
		header[HEADER_CUSTOM_REQUEST_ID] = strconv.FormatUint(requestID, 10)

		_log.Info("SUCCESS: requestID", requestID)
		return responseBuf, header, http.StatusOK, nil
	})

	_log.Debug("SUCCESS ==================================================================================")
}
