package httphandler

import (
    "context"
    "encoding/json"
    "fmt"

    "net/http"
    "reflect"
    "time"

    _ctx "github.com/romapres2010/goapp/pkg/common/ctx"
    _err "github.com/romapres2010/goapp/pkg/common/error"
    _http "github.com/romapres2010/goapp/pkg/common/httpservice"
    _log "github.com/romapres2010/goapp/pkg/common/logger"
    _wp "github.com/romapres2010/goapp/pkg/common/workerpool"
)

type WpFactorialReqResp struct {
    NumArray     []int  `json:"num_array,omitempty"`
    SumFactorial uint64 `json:"sum_factorial,omitempty"`
    Duration     string `json:"duration,omitempty"`
}

// WpHandlerFactorial handle worker pool
func (s *Service) WpHandlerFactorial(w http.ResponseWriter, r *http.Request) {
    _log.Debug("START   ==================================================================================")

    // Запускаем обработчик, возврат ошибки игнорируем
    _ = s.httpService.Process(true, "POST", w, r, func(ctx context.Context, requestBuf []byte, buf []byte) ([]byte, _http.Header, int, error) {
        var requestID = _ctx.FromContextHTTPRequestID(ctx) // RequestID передается через context
        var err error
        var responseBuf []byte
        var wpFactorialReqResp WpFactorialReqResp
        var tic = time.Now()
        var tasks []*_wp.Task

        // Считаем параметры из URL query
        wpTipe := r.URL.Query().Get("wp_tipe")

        _log.Debug("START: requestID", requestID)

        if err = json.Unmarshal(requestBuf, &wpFactorialReqResp); err != nil {
            err = _err.WithCauseTyped(_err.ERR_JSON_UNMARSHAL_ERROR, requestID, err).PrintfError()
            return nil, nil, http.StatusBadRequest, err
        }

        { // Подготовим список задач для запуска
            for i, value := range wpFactorialReqResp.NumArray {
                task := _wp.NewTask(ctx, "CalculateRoute", nil, uint64(i), requestID, s.wpService.GetWPConfig().TaskTimeout, calculateFactorialFn, value)
                tasks = append(tasks, task)
            }

            // в конце обработки отпарить task в кэш для повторного использования
            defer func() {
                for _, task := range tasks {
                    task.Delete()
                }
            }()

        } // Подготовим список задач для запуска

        { // Запускаем обработку
            if wpTipe == "bg" {
                // Запускаем обработку в общий background pool
                _log.Debug("Start with global worker pool: requestID", requestID)
                err = s.wpService.RunTasksGroupWG(requestID, tasks, "Calculate - background")
            } else {
                // Запускаем обработку в локальный пул обработчиков
                _log.Debug("Start with local worker pool: calcId", requestID)
                pool := _wp.NewPool(ctx, requestID, "Calculate - online", s.wpService.GetWPConfig())
                err = pool.RunOnline(requestID, tasks, s.wpService.GetWPConfig().TaskTimeout)
            }

            if err == nil {
                // Суммируем все результаты
                for _, task := range tasks {
                    if task.GetError() == nil {
                        result := task.GetResponses()[0] // ожидаем только один ответ

                        // Приведем к нужному типу
                        if factorial, ok := result.(uint64); ok {
                            wpFactorialReqResp.SumFactorial += factorial
                        } else {
                            err = _err.NewTyped(_err.ERR_INCORRECT_TYPE_ERROR, _err.ERR_UNDEFINED_ID, "WpHandlerFactorial", "0 - uint", reflect.ValueOf(factorial).Type().String(), reflect.ValueOf(uint64(1)).Type().String()).PrintfError()
                            return nil, nil, http.StatusBadRequest, err
                        }
                    } else {
                        return nil, nil, http.StatusBadRequest, task.GetError()
                    }
                }

                wpFactorialReqResp.Duration = fmt.Sprintf("%s", time.Now().Sub(tic))
            } else {
                return nil, nil, http.StatusBadRequest, err
            }
        } // Запускаем обработку

        if responseBuf, err = json.Marshal(wpFactorialReqResp); err != nil {
            err = _err.WithCauseTyped(_err.ERR_JSON_MARSHAL_ERROR, requestID, err).PrintfError()
            return nil, nil, http.StatusBadRequest, err
        }

        // формируем ответ
        header := _http.Header{}
        header[_http.HEADER_CONTENT_TYPE] = _http.HEADER_CONTENT_TYPE_JSON_UTF8
        header[_http.HEADER_CUSTOM_ERR_CODE] = _http.HEADER_CUSTOM_ERR_CODE_SUCCESS

        _log.Debug("SUCCESS", requestID)

        return responseBuf, header, http.StatusOK, nil
    })

    _log.Debug("SUCCESS ==================================================================================")
}

// calculateFactorialFn функция запуска расчета Factorial через worker pool
func calculateFactorialFn(ctx context.Context, data ...interface{}) (error, []interface{}) {
    var factVal uint64 = 1
    var out = make([]interface{}, 1, 1) // ответ содержит только один объект

    if len(data) == 1 {

        // проверяем тип входных параметров
        if value, ok := data[0].(int); !ok {
            return _err.NewTyped(_err.ERR_INCORRECT_TYPE_ERROR, _err.ERR_UNDEFINED_ID, "CalculateFactorialFn", "0 - int", reflect.ValueOf(data[0]).Type().String(), reflect.ValueOf(1).Type().String()).PrintfError(), nil
        } else {

            // Запускаем расчет
            for i := 1; i <= value; i++ {
                factVal *= uint64(i)
                //time.Sleep(time.Millisecond + 10)
            }
        }

        out[0] = factVal // ответ содержит только один объект
        return nil, out  // ошибки расчета транслируем на уровень выше
    }
    return _err.NewTyped(_err.ERR_INCORRECT_ARG_NUM_ERROR, _err.ERR_UNDEFINED_ID, data).PrintfError(), nil
}
