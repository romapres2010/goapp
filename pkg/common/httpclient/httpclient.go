package httpclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"time"

	neturl "net/url"

    _ctx "github.com/romapres2010/goapp/pkg/common/ctx"
    _err "github.com/romapres2010/goapp/pkg/common/error"
    _httplog "github.com/romapres2010/goapp/pkg/common/httplog"
    _http "github.com/romapres2010/goapp/pkg/common/httpservice"
    _log "github.com/romapres2010/goapp/pkg/common/logger"
    _metrics "github.com/romapres2010/goapp/pkg/common/metrics"
)

const HTTP_CLIENT_GET = "GET"
const HTTP_CLIENT_POST = "POST"
const HTTP_CLIENT_PUT = "PUT"
const HTTP_CLIENT_DELETE = "DELETE"

// Header represent temporary HTTP header for save
type Header map[string]string

// Call represent а parameters for client requests
type Call struct {
	URL                string           // URL
	FullURL            string           // URL полный с аргументами
	Args               url.Values       // Аргументы к URL строке
	UserID             string           // UserID для аутентификации
	UserPwd            string           // UserPwd для аутентификации
	CallMethod         string           // HTTP метод для вызова
	CallTimeout        time.Duration    // полный Timeout вызова
	ContentType        string           // тип контента в ответе
	InsecureSkipVerify bool             // игнорировать проверку сертификатов
	ReCallRepeat       int              // количество попыток вызова при недоступности сервиса - 0 не ограничено
	ReCallWaitTimeout  time.Duration    // Timeout между вызовами при недоступности сервиса
	HTTPLogger         *_httplog.Logger // сервис логирования HTTP
}

// Process - represent client common task in process outgoing HTTP request
func (c *Call) Process(ctx context.Context, header Header, requestBody []byte) (statusCode int, responseBuf []byte, respHeader http.Header, err error) {

	var reqID = _http.GetNextHTTPRequestID() // уникальный ID Request
	var req *http.Request
	var resp *http.Response
	var tic = time.Now()

	if c != nil {

		{ // входные проверки
			if ctx == nil {
				return 0, nil, nil, _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, reqID, "if cfg == nil {}").PrintfError()
			}
			if c.URL == "" {
				return 0, nil, nil, _err.NewTyped(_err.ERR_HTTP_CALL_EMPTY_URL, reqID).PrintfError()
			}
		} // входные проверки

		// Добавим метрики
		_metrics.IncHTTPClientCallCountVec(c.URL, c.CallMethod)

		// для каждого запроса поздаем новый контекст, сохраняем в нем уникальный номер HTTP запроса
		callCtx := _ctx.NewContextHTTPRequestID(ctx, reqID)

		// Добавим аргументы к вызову
		if c.Args != nil {
			c.FullURL = c.URL + "?" + url.Values(c.Args).Encode()
		} else {
			c.FullURL = c.URL
		}

		// Создаем новый запрос c контекстом для возможности отмены
		// В тело передаем буфер для передачи в составе запроса
		if requestBody != nil {
			req, err = http.NewRequestWithContext(callCtx, c.CallMethod, c.FullURL, bytes.NewReader(requestBody))
		} else {
			req, err = http.NewRequestWithContext(callCtx, c.CallMethod, c.FullURL, nil)
		}
		if err != nil {
			return 0, nil, nil, _err.WithCauseTyped(_err.ERR_HTTP_CALL_CREATE_CONTEXT_ERROR, reqID, err).PrintfError()
		}

		// запишем заголовок в запрос
		if header != nil {
			for key, h := range header {
				req.Header.Add(key, h)
			}
		}

		// добавим HTTP Basic Authentication
		if c.UserID != "" && c.UserPwd != "" {
			req.SetBasicAuth(c.UserID, c.UserPwd)
		} else {
			_log.Debug("UserID is null or UserPwd is null. Do call without HTTP Basic Authentication: reqID, Method, URL, len(requestBody)", err, reqID, c.CallMethod, c.FullURL, len(requestBody))
		}

		// Скопируем дефолтный транспорт
		tr := http.DefaultTransport.(*http.Transport)

		// переопределим проверку невалидных сертификатов при использовании SSL
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: c.InsecureSkipVerify}

		// создадим HTTP клиента с переопределенным транспортом
		client := &http.Client{
			Transport: tr,
			Timeout:   c.CallTimeout, // полный таймаут ожидания
		}

		// Если сервис не доступен - то цикл с задержкой
		tryCount := 1
		for {
			// проверим, не получен ли сигнал закрытия контекст - останавливаем обработку
			select {
			case <-callCtx.Done():
				return 0, nil, nil, _err.NewTyped(_err.ERR_CONTEXT_CLOSED_ERROR, reqID, "Method, URL, len(requestBody)", []interface{}{c.CallMethod, c.FullURL, len(requestBody)}).PrintfError()
			default:

				// логируем исходящий HTTP запрос
				if c.HTTPLogger != nil {
					_ = c.HTTPLogger.LogHTTPOutRequest(callCtx, req)
				}

				// выполним запрос
				resp, err = client.Do(req)

				// логируем входящий HTTP ответ
				if c.HTTPLogger != nil {
					_ = c.HTTPLogger.LogHTTPInResponse(callCtx, resp)
				}

				// обработаем ошибки
				if err != nil {
					if httperr, ok := err.(*neturl.Error); ok {
						// если прервано по Timeout или произошло закрытие контекста
						if httperr.Timeout() {
							return 0, nil, nil, _err.WithCauseTyped(_err.ERR_CONTEXT_CLOSED_ERROR, reqID, httperr, c.CallTimeout).PrintfError()
						} else {
							return 0, nil, nil, _err.WithCauseTyped(_err.ERR_HTTP_CALL_OTHER_NETURL_ERROR, reqID, httperr, httperr.Error()).PrintfError()
						}
					}
					return 0, nil, nil, _err.WithCauseTyped(_err.ERR_HTTP_CALL_OTHER_ERROR, reqID, err, err.Error()).PrintfError()
				}

				// считаем тело ответа
				if resp.Body != nil {
					responseBuf, err = io.ReadAll(resp.Body)
					defer func(Body io.ReadCloser) {
						err := Body.Close()
						if err != nil {
							_ = _err.WithCauseTyped(_err.ERR_COMMON_ERROR, reqID, err, "httpclient.Process -> io.ReadCloser.Close()").PrintfError()
						}
					}(resp.Body)
					if err != nil {
						return 0, nil, nil, _err.WithCauseTyped(_err.ERR_HTTP_CALL_READ_BODY_ERROR, reqID, err, err.Error()).PrintfError()
					}
				}

				// Добавим метрики
				_metrics.IncHTTPClientCallDurationVec(c.URL, c.CallMethod, time.Now().Sub(tic))

				_log.Debug("Process HTTP call: reqID, Method, URL, len(requestBody), resp.StatusCode, len(responseBuf), duration", reqID, c.CallMethod, c.FullURL, len(requestBody), resp.Status, len(responseBuf), time.Now().Sub(tic))

				// частичный анализ статуса ответа
				if resp.StatusCode == http.StatusNotFound {
					// Если превышено количество попыток то на выход
					if c.ReCallRepeat != 0 && tryCount >= c.ReCallRepeat {
						return http.StatusNotFound, nil, resp.Header, _err.NewTyped(_err.ERR_HTTP_CALL_URL_NOT_FOUND_ERROR, reqID, c.ReCallRepeat, c.URL, c.Args).PrintfError()
					}

					// Если URL не доступен - продолжаем в цикле
					_log.Info("URL was not found, wait and try again: : reqID, Method, URL, ReCallWaitTimeout, tryCount", reqID, c.CallMethod, c.FullURL, c.ReCallWaitTimeout, tryCount)

					// делаем задержку
					time.Sleep(time.Duration(c.ReCallWaitTimeout))
					tryCount++

					break // выходим на начало цикла
				} else if resp.StatusCode == http.StatusMethodNotAllowed {
					return http.StatusMethodNotAllowed, nil, resp.Header, _err.NewTyped(_err.ERR_HTTP_CALL_METHOD_NOT_ALLOWED_ERROR, reqID, c.CallMethod).PrintfError()
				} else if resp.StatusCode == http.StatusBadRequest {
					_log.Error("Process HTTP call - StatusBadRequest: reqID, Method, URL, len(requestBody), resp.StatusCode, len(responseBuf), duration", reqID, c.CallMethod, c.FullURL, len(requestBody), resp.Status, len(responseBuf), time.Now().Sub(tic))
				}

				// возврат на уровень вверх для дальнейшего анализа ответа
				return resp.StatusCode, responseBuf, resp.Header, nil
			}
		}
	}
	return 0, nil, nil, _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, reqID, "c != nil").PrintfError()
}
