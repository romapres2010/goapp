package httplog

import (
	"context"
	"fmt"

	"net/http"
	"net/http/httputil"

	"gopkg.in/natefinch/lumberjack.v2"

	_ctx "github.com/romapres2010/goapp/pkg/common/ctx"
	_err "github.com/romapres2010/goapp/pkg/common/error"
	_log "github.com/romapres2010/goapp/pkg/common/logger"
)

const LOG_DEF_FILE_NAME = "./httplog/http.log"

// Logger represent аn HTTP logger
type Logger struct {
	fileLogger *lumberjack.Logger
	cfg        *Config // конфигурационные параметры
}

// Config represent аn HTTP logger config
type Config struct {
	Enable     bool `yaml:"enable" json:"enable"`             // состояние логирования 'true', 'false'
	LogInReq   bool `yaml:"log_in_req" json:"log_in_req"`     // логировать входящие запросы
	LogOutReq  bool `yaml:"log_out_req" json:"log_out_req"`   // логировать исходящие запросы
	LogInResp  bool `yaml:"log_in_resp" json:"log_in_resp"`   // логировать входящие ответы
	LogOutResp bool `yaml:"log_out_resp" json:"log_out_resp"` // логировать исходящие ответы
	LogBody    bool `yaml:"log_body" json:"log_body"`         // логировать тело запроса

	FileConfig *_log.FileConfig `json:"file" yaml:"file"` // конфигурация для логирования в файл, если имя файла пустое, то используется глобальное
}

// New - создает новый Logger
func New(ctx context.Context, cfg *Config) (*Logger, error) {
	var err error
	var requestID = _ctx.FromContextHTTPRequestID(ctx) // RequestID передается через context

	_log.Info("Creating new HTTP traffic logger")

	{ // входные проверки
		if cfg == nil {
			return nil, _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, requestID, "if cfg == nil {}").PrintfError()
		}
	} // входные проверки

	logger := Logger{}

	if err = logger.SetConfig(cfg); err != nil {
		return nil, err
	} else {
		return &logger, nil
	}
}

// GetConfig get logger config
func (log *Logger) GetConfig() *Config {
	if log != nil {
		return log.cfg
	}
	return nil
}

// SetConfig set new logger config
func (log *Logger) SetConfig(cfg *Config) error {
	if cfg != nil {
		var err error

		_log.Info("Set HTTP logger config: cfg", *cfg)

		//Если ранее логирование было включено и его выключают, то закрыть файл
		if log.fileLogger != nil {
			if err = log.fileLogger.Close(); err != nil {
				return err
			}
		}

		log.cfg = cfg // новый конфиг

		if log.cfg.Enable {
			if cfg.FileConfig.Filename == "" {
				cfg.FileConfig.Filename = LOG_DEF_FILE_NAME
			}
			log.fileLogger = _log.NewFileLogger(*cfg.FileConfig, "")
		}
	}

	return nil
}

// Shutdown shutting down service
func (log *Logger) Shutdown() (myerr error) {
	_log.Info("Shutting down HTTP log service")

	if log.fileLogger != nil {
		return log.fileLogger.Close()
	} else {
		return nil
	}
}

// LogHTTPOutRequest process HTTP logging for Out request
func (log *Logger) LogHTTPOutRequest(ctx context.Context, req *http.Request) error {
	if log.cfg.Enable && log.fileLogger != nil {
		requestID := _ctx.FromContextHTTPRequestID(ctx) // RequestID передается через context

		_log.Debug("Logging HTTP out request: requestID", requestID)

		if req != nil && log.cfg.LogOutReq {
			dump, err := httputil.DumpRequestOut(req, log.cfg.LogBody)
			if err != nil {
				return _err.WithCauseTyped(_err.ERR_HTTP_DUMP_REQUEST_ERROR, requestID, err, err.Error()).PrintfError()
			}
			_, _ = fmt.Fprintf(log.fileLogger, "'%s' Out Request '%v' BEGIN ==================================================================== \n", _log.GetTimestampStr(), requestID)
			_, _ = fmt.Fprintf(log.fileLogger, "%+v\n", string(dump))
			_, _ = fmt.Fprintf(log.fileLogger, "'%s' Out Request '%v' END ==================================================================== \n", _log.GetTimestampStr(), requestID)
		}
	}
	return nil
}

// LogHTTPInResponse process HTTP logging for In response
func (log *Logger) LogHTTPInResponse(ctx context.Context, resp *http.Response) error {
	if log.cfg.Enable && log.fileLogger != nil {

		requestID := _ctx.FromContextHTTPRequestID(ctx) // RequestID передается через context

		_log.Debug("Logging HTTP in response: requestID", requestID)

		if resp != nil && log.cfg.LogInResp {
			dump, err := httputil.DumpResponse(resp, log.cfg.LogBody)
			if err != nil {
				return _err.WithCauseTyped(_err.ERR_HTTP_DUMP_REQUEST_ERROR, requestID, err, err.Error()).PrintfError()
			}
			_, _ = fmt.Fprintf(log.fileLogger, "'%s' In Response '%v' BEGIN ==================================================================== \n", _log.GetTimestampStr(), requestID)
			_, _ = fmt.Fprintf(log.fileLogger, "%+v\n", string(dump))
			_, _ = fmt.Fprintf(log.fileLogger, "'%s' In Response '%v' End ==================================================================== \n", _log.GetTimestampStr(), requestID)
		}
	}
	return nil
}

// LogHTTPInRequest process HTTP logging for In request
func (log *Logger) LogHTTPInRequest(ctx context.Context, req *http.Request) error {
	if log.cfg.Enable && log.fileLogger != nil {

		requestID := _ctx.FromContextHTTPRequestID(ctx) // RequestID передается через context

		_log.Debug("Logging HTTP in request: requestID", requestID)

		if req != nil && log.cfg.LogInReq {
			dump, err := httputil.DumpRequest(req, log.cfg.LogBody)
			if err != nil {
				return _err.WithCauseTyped(_err.ERR_HTTP_DUMP_REQUEST_ERROR, requestID, err, err.Error()).PrintfError()
			}
			_, _ = fmt.Fprintf(log.fileLogger, "'%s' In Request '%v' BEGIN ==================================================================== \n", _log.GetTimestampStr(), requestID)
			_, _ = fmt.Fprintf(log.fileLogger, "%+v\n", string(dump))
			_, _ = fmt.Fprintf(log.fileLogger, "'%s' In Request '%v' End ==================================================================== \n", _log.GetTimestampStr(), requestID)
		}
	}
	return nil
}

// LogHTTPOutResponse process HTTP logging for Out Response
func (log *Logger) LogHTTPOutResponse(ctx context.Context, header map[string]string, responseBuf []byte, status int) error {
	if log.cfg.Enable && log.fileLogger != nil && log.cfg.LogOutResp {

		requestID := _ctx.FromContextHTTPRequestID(ctx) // RequestID передается через context

		_log.Debug("Logging HTTP out response: requestID", requestID)

		// сформируем буфер с ответом
		dump := make([]byte, 0)

		// добавим статус ответа
		dump = append(dump, []byte(fmt.Sprintf("HTTP %v %s\n", status, http.StatusText(status)))...)

		// соберем все заголовки в буфер для логирования
		if header != nil {
			for k, v := range header {
				dump = append(dump, []byte(fmt.Sprintf("%s: %s\n", k, v))...)
			}
		}

		// Добавим в буфер тело
		if log.cfg.LogBody && responseBuf != nil {
			dump = append(dump, []byte("\n")...)
			dump = append(dump, responseBuf...)
		}

		_, _ = fmt.Fprintf(log.fileLogger, "'%s' Out Response '%v' BEGIN ==================================================================== \n", _log.GetTimestampStr(), requestID)
		_, _ = fmt.Fprintf(log.fileLogger, "%+v\n", string(dump))
		_, _ = fmt.Fprintf(log.fileLogger, "'%s' Out Response '%v' End ==================================================================== \n", _log.GetTimestampStr(), requestID)
	}
	return nil
}
