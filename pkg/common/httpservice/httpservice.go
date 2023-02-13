package httpservice

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "reflect"
    "strconv"
    "strings"
    "sync/atomic"
    "time"

    "encoding/json"

    bytespool "github.com/romapres2010/goapp/pkg/common/bytespool"
    _ctx "github.com/romapres2010/goapp/pkg/common/ctx"
    _err "github.com/romapres2010/goapp/pkg/common/error"
    _httplog "github.com/romapres2010/goapp/pkg/common/httplog"
    _jwt "github.com/romapres2010/goapp/pkg/common/jwt"
    _log "github.com/romapres2010/goapp/pkg/common/logger"
    _metrics "github.com/romapres2010/goapp/pkg/common/metrics"
    _recover "github.com/romapres2010/goapp/pkg/common/recover"
)

const (
    HEADER_CUSTOM_ERR_CODE                  = "Custom-Err-Code"
    HEADER_CUSTOM_ERR_CODE_SUCCESS          = "0"
    HEADER_CUSTOM_ERR_MESSAGE               = "Custom-Err-Message"
    HEADER_CUSTOM_ERR_CAUSE_MESSAGE         = "Custom-Err-Cause-Message"
    HEADER_CUSTOM_ERR_TRACE                 = "Custom-Err-Trace"
    HEADER_CUSTOM_ID                        = "Custom-Id"
    HEADER_CUSTOM_REQUEST_ID                = "Custom-Request-Id"
    HEADER_CONTENT_TYPE                     = "Content-Type"
    HEADER_CONTENT_TYPE_PLAIN_UTF8          = "text/plain; charset=utf-8"
    HEADER_CONTENT_TYPE_JSON_UTF8           = "application/json; charset=utf-8"
    HEADER_CONTENT_TYPE_XML_UTF8            = "application/xml; charset=utf-8"
    HEADER_CONTENT_TYPE_OCTET_STREAM        = "application/octet-stream"
    HEADER_CONTENT_DISPOSITION              = "Content-Disposition"
    HEADER_CONTENT_TRANSFER_ENCODING        = "Content-Transfer-Encoding"
    HEADER_CONTENT_TRANSFER_ENCODING_BINARY = "binary"
    HEADER_LOG_LEVEL_GLOBALLOGFILTER        = "Log-Level-globalLogFilter"
    HEADER_LOG_HTTP_TRAFFIC_STATUS          = "Log-HTTP-Traffic-Status"
    HEADER_LOG_HTTP_TRAFFIC_TYPE            = "Log-HTTP-Traffic-Type"
    HEADER_LOG_ERROR_TO_HTTP_TYPE           = "Log-Error-To-HTTP-Type"

    AUTH_TYPE_NONE     = "NONE"
    AUTH_TYPE_INTERNAL = "INTERNAL"
    AUTH_TYPE_MSAD     = "MSAD"
)

// Header represent temporary HTTP header
type Header map[string]string
type HandleFunc func(http.ResponseWriter, *http.Request)

// Handler represent HTTP handler
type Handler struct {
    Path        string
    HandlerFunc http.HandlerFunc
    Method      string
}

// Handlers represent HTTP handlers map
type Handlers map[string]Handler

// уникальный номер HTTP запроса
var httpRequestID uint64

// GetNextHTTPRequestID - запросить номер следующего HTTP запроса
func GetNextHTTPRequestID() uint64 {
    return atomic.AddUint64(&httpRequestID, 1)
}

// Service represent HTTP service
type Service struct {
    ctx    context.Context    // корневой контекст при инициации сервиса
    cancel context.CancelFunc // функция закрытия глобального контекста
    cfg    *Config            // конфигурационные параметры

    Handlers Handlers // список обработчиков

    // вложенные сервисы
    httpHandler interface{}      // обработчики HTTP
    httpLogger  *_httplog.Logger // сервис логирования HTTP
    bytesPool   *bytespool.Pool  // represent pooling of []byte
}

// HandlerConfig represent HTTP handler configurations
type HandlerConfig struct {
    Enabled     bool   `yaml:"enabled" json:"enabled"`                     // Признак включен ли сервис
    Application string `yaml:"application" json:"application"`             // Приложение к которому относится сервис, например, app
    Module      string `yaml:"module" json:"module,omitempty"`             // Модуль к которому относится сервис, например, calculator
    Service     string `yaml:"service" json:"service,omitempty"`           // Имя сервиса, например, berth
    Version     string `yaml:"version" json:"version,omitempty"`           // Версия сервиса
    FullPath    string `yaml:"full_path" json:"full_path,omitempty"`       // URI сервиса /Application.Module.Service.APIVersion или /Application/APIVersion/Module/Service
    Params      string `yaml:"params" json:"params,omitempty"`             // Параметры сервиса с виде {id:[0-9]+}
    Method      string `yaml:"method" json:"method,omitempty"`             // HTTP метод: GET, POST, ...
    HandlerName string `yaml:"handler_name" json:"handler_name,omitempty"` // Имя функции обработчика
}

// Config represent HTTP Service configurations
type Config struct {
    JwtKeyStr          string `yaml:"-" json:"-"`                                               // JWT secret key
    JwtKey             []byte `yaml:"-" json:"-"`                                               // JWT secret key
    HTTPUser           string `yaml:"-" json:"-"`                                               // пользователь для HTTP Basic Authentication передается через командую строку
    HTTPPass           string `yaml:"-" json:"-"`                                               // пароль для HTTP Basic Authentication передается через командую строку
    AuthType           string `yaml:"auth_type" json:"auth_type"`                               // Authentication type NONE, INTERNAL, MSAD
    MaxBodyBytes       int    `yaml:"max_body_bytes" json:"max_body_bytes"`                     // HTTP max body bytes - default 0 - unlimited
    UseHSTS            bool   `yaml:"use_hsts" json:"use_hsts"`                                 // use HTTP Strict Transport Security
    UseJWT             bool   `yaml:"use_jwt" json:"use_jwt"`                                   // use JSON web token (JWT)
    JWTExpiresAt       int    `yaml:"jwt_expires_at" json:"jwt_expires_at"`                     // JWT expiry time in seconds - 0 without restriction
    MSADServer         string `yaml:"msad_server" json:"msad_server"`                           // MS Active Directory server
    MSADPort           int    `yaml:"msad_port" json:"msad_port"`                               // MS Active Directory Port
    MSADBaseDN         string `yaml:"msad_base_dn" json:"msad_base_dn"`                         // MS Active Directory BaseDN
    MSADSecurity       int    `yaml:"msad_security" json:"msad_security"`                       // MS Active Directory Security: SecurityNone, SecurityTLS, SecurityStartTLS
    UseBufPool         bool   `yaml:"use_buf_pool" json:"use_buf_pool"`                         // use byte polling for JSON -> HTTP
    BufPooledSize      int    `yaml:"buf_pooled_size" json:"buf_pooled_size"`                   // recommended size of polling for JSON -> HTTP
    BufPooledMaxSize   int    `yaml:"buf_pooled_max_size" json:"buf_pooled_max_size"`           // max size of polling for JSON -> HTTP
    HTTPErrorLogHeader bool   `yaml:"log_error_to_http_header" json:"log_error_to_http_header"` // log any error to HTTP response header
    HTTPErrorLogBody   bool   `yaml:"log_error_to_http_body" json:"log_error_to_http_body"`     // log any error to HTTP response body
    HTTPHeaderMaxSize  int    `yaml:"http_header_max_size" json:"http_header_max_size"`         // max size HTTP header element - use for out response

    Handlers map[string]HandlerConfig `yaml:"handlers" json:"handlers"` // HTTP handler configurations

    // конфигурация вложенных сервисов
    BytesPoolCfg bytespool.Config `yaml:"-" json:"-"` // конфигурация bytesPool
}

// New create new HTTP service
func New(ctx context.Context, cfg *Config, httpLogger *_httplog.Logger) (*Service, *_httplog.Logger, error) {

    requestID := _ctx.FromContextHTTPRequestID(ctx) // RequestID передается через context

    _log.Info("Creating new HTTP service")

    { // входные проверки
        if cfg == nil {
            return nil, nil, _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, requestID, "if cfg == nil {}").PrintfError()
        }
        if httpLogger == nil {
            return nil, nil, _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, requestID, "if httpLogger == nil {}").PrintfError()
        }
    } // входные проверки

    service := &Service{
        cfg:        cfg,
        httpLogger: httpLogger,
    }

    // создаем контекст с отменой
    if ctx == nil {
        service.ctx, service.cancel = context.WithCancel(context.Background())
    } else {
        service.ctx, service.cancel = context.WithCancel(ctx)
    }

    // Наполним список системных обработчиков
    service.Handlers = map[string]Handler{
        "EchoHandler":            {"/system/echo", service.recoverWrap(service.EchoHandler), "POST"},
        "SingingHandler":         {"/system/auth/signin", service.recoverWrap(service.SingingHandler), "POST"},
        "JWTRefreshHandler":      {"/system/auth/refresh", service.recoverWrap(service.JWTRefreshHandler), "POST"},
        "HTTPLogHandler":         {"/system/config/httplog", service.recoverWrap(service.HTTPLogHandler), "POST"},
        "HTTPErrorLogHandler":    {"/system/config/httperrlog", service.recoverWrap(service.HTTPErrorLogHandler), "POST"},
        "LogLevelHandler":        {"/system/config/loglevel", service.recoverWrap(service.LogLevelHandler), "POST"},
        "LoggerGetConfigHandler": {"/system/config/logger", service.recoverWrap(service.LoggerGetConfigHandler), "GET"},
        "LoggerSetConfigHandler": {"/system/config/logger", service.recoverWrap(service.LoggerSetConfigHandler), "POST"}}

    // создаем BytesPool
    if service.cfg.UseBufPool {
        service.cfg.BytesPoolCfg.PooledSize = service.cfg.BufPooledSize
        service.bytesPool = bytespool.New(&service.cfg.BytesPoolCfg)
    }

    _log.Info("HTTP service was created")
    return service, service.httpLogger, nil
}

// SetHttpHandler create HTTP handlers
func (s *Service) SetHttpHandler(ctx context.Context, httpHandler interface{}) (err error) {
    requestID := _ctx.FromContextHTTPRequestID(ctx) // RequestID передается через context

    _log.Info("Register HTTP handlers")

    { // входные проверки
        if httpHandler == nil {
            return _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, requestID, "if httpHandler == nil {}").PrintfError()
        }
    } // входные проверки

    s.httpHandler = httpHandler

    { // добавим обработчиков из конфига
        var dummyHandlerFunc func(http.ResponseWriter, *http.Request)
        for handlerName, handlerCfg := range s.cfg.Handlers {
            if handlerCfg.Enabled { // сервис включен
                handler := Handler{}

                if handlerCfg.Params != "" {
                    handler.Path = handlerCfg.FullPath + "/" + handlerCfg.Params
                } else {
                    handler.Path = handlerCfg.FullPath
                }
                handler.Method = handlerCfg.Method

                // Определим метод обработчика
                method := reflect.ValueOf(s.httpHandler).MethodByName(handlerCfg.HandlerName)

                // Метод найден
                if method.IsValid() {
                    methodInterface := method.Interface()                                         // получил метод в виде интерфейса, для дальнейшего преобразования к нужному типу
                    handlerFunc, ok := methodInterface.(func(http.ResponseWriter, *http.Request)) // преобразуем к нужному типу
                    if ok {
                        handler.HandlerFunc = s.recoverWrap(handlerFunc) // Оборачиваем от паники
                        _log.Info("Register HTTP handler: HandlerName, Method, FullPath", handlerCfg.HandlerName, handlerCfg.Method, handlerCfg.FullPath)
                    } else {
                        return _err.NewTyped(_err.ERR_INCORRECT_TYPE_ERROR, requestID, "New", "func(http.ResponseWriter, *http.Request)", reflect.ValueOf(methodInterface).Type().String(), reflect.ValueOf(dummyHandlerFunc).Type().String()).PrintfError()
                    }
                } else {
                    return _err.NewTyped(_err.ERR_HTTP_HANDLER_METHOD_NOT_FOUND, requestID, handlerCfg.HandlerName).PrintfError()
                }

                s.Handlers[handlerName] = handler
            }
        }
    } // добавим обработчиков из конфига

    return nil
}

// Shutdown shutting down service
func (s *Service) Shutdown() (err error) {
    _log.Info("Shutting down HTTP service")

    s.cancel() // закрываем контекст

    // Print statistics about bytes pool
    if s.cfg.UseBufPool && s.bytesPool != nil {
        s.bytesPool.PrintBytesPoolStats()
    }

    return nil
}

// recoverWrap cover handler functions with panic recover
func (s *Service) recoverWrap(handlerFunc http.HandlerFunc) http.HandlerFunc {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // функция восстановления после паники
        defer func() {
            rec := recover()
            if r != nil {
                requestID := _ctx.FromContextHTTPRequestID(s.ctx) // ID передается через context
                err := _recover.GetRecoverError(rec, requestID)
                if err != nil {
                    // расширенное логирование ошибки в контексте HTTP
                    _log.Info("HTTP Handler recover from panic: r", r)
                    s.processError(err, w, r, http.StatusInternalServerError, requestID)
                }
            }
        }()

        // вызываем обработчик
        if handlerFunc != nil {
            tic := time.Now()

            // подсчет активных сессий
            _metrics.IncHTTPActiveRequestsCount()
            defer _metrics.DecHTTPActiveRequestsCount()

            handlerFunc(w, r)

            _metrics.AddHTTPRequestsDuration(time.Now().Sub(tic))
        }
    })
}

// Process - represent server common task in Process incoming HTTP request
func (s *Service) Process(ignoreBufPool bool, method string, w http.ResponseWriter, r *http.Request, fn func(ctx context.Context, requestBuf []byte, buf []byte) ([]byte, Header, int, error)) (myerr error) {

    var responseBuf []byte
    var header Header
    var status int
    var respWrittenLen int
    var tic = time.Now()

    // Получить уникальный номер HTTP запроса
    requestID := GetNextHTTPRequestID()

    // для каждого запроса создаем новый контекст, сохраняем в нем уникальный номер HTTP запроса
    ctx := _ctx.NewContextHTTPRequestID(s.ctx, requestID)

    // Логируем входящий HTTP запрос
    if s.httpLogger != nil {
        _ = s.httpLogger.LogHTTPInRequest(ctx, r) // При сбое HTTP логирования, делаем системное логирование, но работу не останавливаем
    }

    // Добавим метрики
    _metrics.IncHTTPRequestsCountVec(r.RequestURI, r.Method)

    // Проверим разрешенный метод
    _log.Debug("Check allowed HTTP method: requestID, request.Method, method", requestID, r.Method, method)
    if r.Method != method {
        myerr = _err.NewTyped(_err.ERR_HTTP_METHOD_NOT_ALLOWED_ERROR, requestID, r.Method, method).PrintfError()
        s.processError(myerr, w, r, http.StatusMethodNotAllowed, requestID) // расширенное логирование ошибки в контексте HTTP
        return myerr
    }

    // Если включен режим аутентификации без использования JWT токена, то проверять пользователя и пароль каждый раз
    _log.Debug("Check authentication method: requestID, AuthType", requestID, s.cfg.AuthType)
    if (s.cfg.AuthType == AUTH_TYPE_INTERNAL || s.cfg.AuthType == AUTH_TYPE_MSAD) && !s.cfg.UseJWT {
        _log.Debug("JWT is off. Need Authentication: requestID", requestID)

        // Считаем из заголовка HTTP Basic Authentication
        username, password, ok := r.BasicAuth()
        if !ok {
            myerr = _err.NewTyped(_err.ERR_HTTP_AUTH_BASIC_NOT_SET_ERROR, requestID).PrintfError()
            s.processError(myerr, w, r, http.StatusUnauthorized, requestID)
            return myerr
        }
        _log.Debug("Get Authorization header: username", username)

        // Выполняем аутентификацию
        if myerr = s.checkAuthentication(requestID, username, password); myerr != nil {
            _log.ErrorAsInfo(myerr)
            s.processError(myerr, w, r, http.StatusUnauthorized, requestID)
            return myerr
        }
    }

    // Если используем JWT - проверим токен
    if s.cfg.UseJWT {
        _log.Debug("JWT is on. Check JSON web token: requestID", requestID)

        // Считаем token из requests cookies
        cookie, err := r.Cookie("token")
        if err != nil {
            myerr = _err.WithCauseTyped(_err.ERR_HTTP_AUTH_JWT_NOT_SET_ERROR, requestID, err).PrintfError()
            s.processError(myerr, w, r, http.StatusUnauthorized, requestID) // расширенное логирование ошибки в контексте HTTP
            return myerr
        }

        // Проверим JWT в token
        if myerr = _jwt.CheckJWTFromCookie(requestID, cookie, s.cfg.JwtKey); myerr != nil {
            _log.ErrorAsInfo(myerr)
            s.processError(myerr, w, r, http.StatusUnauthorized, requestID) // расширенное логирование ошибки в контексте HTTP
            return myerr
        }
    }

    // Считаем тело запроса
    _log.Debug("Reading request body: requestID", requestID)
    requestBuf, err := io.ReadAll(r.Body)
    if err != nil {
        myerr = _err.WithCauseTyped(_err.ERR_HTTP_BODY_READ_ERROR, requestID, err).PrintfError()
        s.processError(myerr, w, r, http.StatusInternalServerError, requestID) // расширенное логирование ошибки в контексте HTTP
        return myerr
    }
    _log.Debug("Read request body: requestID, len(body)", requestID, len(requestBuf))

    // Выделяем новый буфер из pool, он может использоваться для копирования JSON / XML
    // Если буфер будет недостаточного размера, то он не будет использован
    var buf []byte
    if !ignoreBufPool && s.cfg.UseBufPool && s.bytesPool != nil {
        buf = s.bytesPool.GetBuf()
        _log.Debug("Got []byte buffer from pool: size", cap(buf))
    }

    // вызываем обработчик
    _log.Debug("Calling external function handler: requestID, function", requestID, reflect.ValueOf(fn).String())
    responseBuf, header, status, myerr = fn(ctx, requestBuf, buf)
    if myerr != nil {
        if responseBuf != nil {
            s.processError(myerr, w, r, status, requestID, responseBuf) // расширенное логирование ошибки в контексте HTTP
        } else {
            s.processError(myerr, w, r, status, requestID) // расширенное логирование ошибки в контексте HTTP
        }
        return myerr
    }

    // Если переданного буфера не хватило, то мог быть создан новый буфер. Вернем его в pool
    if !ignoreBufPool && responseBuf != nil && buf != nil && s.cfg.UseBufPool && s.bytesPool != nil {
        // Если новый буфер подходит по размерам для хранения в pool
        if cap(responseBuf) >= s.cfg.BufPooledSize && cap(responseBuf) <= s.cfg.BufPooledMaxSize {
            defer s.bytesPool.PutBuf(responseBuf)
        }

        if reflect.ValueOf(buf).Pointer() != reflect.ValueOf(responseBuf).Pointer() {
            _log.Debug("[]byte buffer: poolBufSize, responseBufSize", cap(buf), cap(responseBuf))
        }
    }

    // Логируем ответ в файл
    if s.httpLogger != nil {
        _ = s.httpLogger.LogHTTPOutResponse(ctx, header, responseBuf, status) // При сбое HTTP логирования, делаем системное логирование, но работу не останавливаем
    }

    // Записываем заголовок ответа
    _log.Debug("Set HTTP response headers: requestID", requestID)
    if header != nil {
        for key, h := range header {
            w.Header().Set(key, h)
        }
    }

    // Устанавливаем HSTS Strict-Transport-Security
    if s.cfg.UseHSTS {
        _log.Debug("Set HSTS Strict-Transport-Security header")
        w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
    }

    // Записываем HTTP статус ответа
    _log.Debug("Set HTTP response status: requestID, Status", requestID, http.StatusText(status))
    w.WriteHeader(status)

    _metrics.IncHTTPRequestsDurationVec(r.RequestURI, r.Method, "BEFORE WRITE", time.Now().Sub(tic))

    // Записываем тело ответа
    if responseBuf != nil && len(responseBuf) > 0 {
        _log.Debug("Writing HTTP response body: requestID, len(body)", requestID, len(responseBuf))
        respWrittenLen, err = w.Write(responseBuf)
        if err != nil {
            myerr = _err.WithCauseTyped(_err.ERR_HTTP_REQUEST_WRITE_ERROR, requestID, err).PrintfError()
            s.processError(myerr, w, r, http.StatusInternalServerError, requestID) // расширенное логирование ошибки в контексте HTTP
            return myerr
        }
        _log.Debug("Written HTTP response: requestID, len(body)", requestID, respWrittenLen)
    } else {
        _log.Debug("HTTP response body is empty")
    }

    // Добавим метрики
    _metrics.IncHTTPRequestsDurationVec(r.RequestURI, r.Method, "FINAL", time.Now().Sub(tic))

    return nil
}

func substrRune(str string, from int, to int) string {
    if len(str) > 0 {
        runes := []rune(str) // convert string to rune slice
        fromPos := from
        toPos := to

        if fromPos < 0 {
            fromPos = 0
        }
        if toPos > len(runes) {
            toPos = len(runes) - 1
        }
        if fromPos > len(runes) {
            fromPos = len(runes) - 1
        }
        if toPos < fromPos {
            toPos = fromPos
        }

        return string(runes[fromPos:toPos]) // take subslice of rune and convert back to string
    } else {
        return str
    }
}

// processError - log error into header and body
func (s *Service) processError(err error, w http.ResponseWriter, r *http.Request, status int, requestID uint64, bufs ...[]byte) {

    // логируем в файл с полной трассировкой
    _log.Error(fmt.Sprintf("requestID:['%v'], %v", requestID, err))

    if w != nil && r != nil && err != nil {
        var myerr *_err.Error
        var isMyErr bool

        _metrics.IncHTTPErrorRequestsCountVec(r.RequestURI, r.Method)

        // Запишем базовые заголовки
        w.Header().Set(HEADER_CONTENT_TYPE, HEADER_CONTENT_TYPE_JSON_UTF8)
        w.Header().Set(HEADER_CUSTOM_REQUEST_ID, strconv.FormatUint(requestID, 10))

        if s.cfg.HTTPErrorLogHeader {
            // Заменим в заголовке запрещенные символы на пробел
            // carriage return (CR, ASCII 0xd), line feed (LF, ASCII 0xa), and the zero character (NUL, ASCII 0x0)
            headerReplacer := strings.NewReplacer("\x0a", " ", "\x0d", " ", "\x00", " ")

            // Запишем текст ошибки в заголовок ответа
            if myerr, isMyErr = err.(*_err.Error); isMyErr {
                // если тип ошибки _err.Error, то возьмем коды из нее
                w.Header().Set(HEADER_CUSTOM_ERR_CODE, substrRune(headerReplacer.Replace(myerr.Code), 0, s.cfg.HTTPHeaderMaxSize))
                w.Header().Set(HEADER_CUSTOM_ERR_MESSAGE, substrRune(headerReplacer.Replace(fmt.Sprintf("%v", myerr)), 0, s.cfg.HTTPHeaderMaxSize))
                w.Header().Set(HEADER_CUSTOM_ERR_CAUSE_MESSAGE, substrRune(headerReplacer.Replace(myerr.CauseMessage), 0, s.cfg.HTTPHeaderMaxSize))
                w.Header().Set(HEADER_CUSTOM_ERR_TRACE, substrRune(headerReplacer.Replace(myerr.Trace), 0, s.cfg.HTTPHeaderMaxSize))
            } else {
                w.Header().Set(HEADER_CUSTOM_ERR_MESSAGE, substrRune(headerReplacer.Replace(fmt.Sprintf("%+v", err)), 0, s.cfg.HTTPHeaderMaxSize))
            }
        }

        w.WriteHeader(status) // Запишем статус ответа

        // Запишем ошибку в тело ответа
        if s.cfg.HTTPErrorLogBody {
            type ErrorResponse struct {
                Error          error             `json:"__ERROR__,omitempty"`
                ResponseJson   []json.RawMessage `json:"response_json,omitempty"`
                ResponseString []string          `json:"response_str,omitempty"`
            }

            var valJson []byte
            var errorResponse *ErrorResponse

            if !isMyErr {
                errorResponse = &ErrorResponse{Error: error(err)}
            } else {
                myerr.Trace = ""
                myerr.Args = ""
                errorResponse = &ErrorResponse{Error: myerr}
            }

            if bufs != nil && len(bufs) > 0 {
                for _, buf := range bufs {
                    if len(buf) > 0 {
                        if json.Valid(buf) {
                            errorResponse.ResponseJson = append(errorResponse.ResponseJson, buf)
                        } else {
                            errorResponse.ResponseString = append(errorResponse.ResponseString, string(buf))
                        }
                    }
                }
            }

            valJson, err = json.MarshalIndent(errorResponse, "", "    ")
            if err != nil {
                _, _ = fmt.Fprintln(w, err.Error())
                _, _ = fmt.Fprintln(w, errorResponse.Error.Error())
            } else {
                _, _ = fmt.Fprintln(w, string(valJson))
            }

            // TODO - добавить логирование в файл ошибочных ответов
            //// Логируем ответ в файл
            //if s.httpLogger != nil {
            //	_ = s.httpLogger.LogHTTPOutResponse(ctx, header, responseBuf, status) // При сбое HTTP логирования, делаем системное логирование, но работу не останавливаем
            //}

        }
    }
}

func UrlValuesToMap(values url.Values) map[string]string {
    res := make(map[string]string)
    for key, value := range values {
        if len(value) > 0 {
            res[key] = value[0]
        }
    }
    return res
}
