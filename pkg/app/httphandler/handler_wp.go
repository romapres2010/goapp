package httphandler

import (
	"context"
	"encoding/json"
	_wpservice "github.com/romapres2010/goapp/pkg/common/workerpoolservice"
	"net/http"
	"reflect"

	_ctx "github.com/romapres2010/goapp/pkg/common/ctx"
	_err "github.com/romapres2010/goapp/pkg/common/error"
	_http "github.com/romapres2010/goapp/pkg/common/httpservice"
	_wp "github.com/romapres2010/goapp/pkg/common/workerpool"
)

type WpFactorialReqResp struct {
	NumArray     []uint64 `json:"num_array,omitempty"`
	SumFactorial uint64   `json:"sum_factorial,omitempty"`
	Duration     string   `json:"duration,omitempty"`
}

// WpHandlerFactorial handle worker pool
func (s *Service) WpHandlerFactorial(w http.ResponseWriter, r *http.Request) {
	//_log.Debug("START   ==================================================================================")

	// Запускаем обработчик, возврат ошибки игнорируем
	_ = s.httpService.Process(true, "POST", w, r, func(ctx context.Context, requestBuf []byte, buf []byte) ([]byte, _http.Header, int, error) {
		var requestID = _ctx.FromContextHTTPRequestID(ctx) // RequestID передается через context
		var err error
		var responseBuf []byte
		var wpFactorialReqResp WpFactorialReqResp

		// Считаем параметры из URL query
		wpTipe := r.URL.Query().Get("wp_tipe")

		//_log.Debug("START: requestID", requestID)

		if err = json.Unmarshal(requestBuf, &wpFactorialReqResp); err != nil {
			err = _err.WithCauseTyped(_err.ERR_JSON_UNMARSHAL_ERROR, requestID, err).PrintfError()
			return nil, nil, http.StatusBadRequest, err
		}

		// Запускаем обработку
		err = calculateFactorial(ctx, s.wpService, requestID, &wpFactorialReqResp, wpTipe)
		if err != nil {
			return nil, nil, http.StatusBadRequest, err
		}

		if responseBuf, err = json.Marshal(wpFactorialReqResp); err != nil {
			err = _err.WithCauseTyped(_err.ERR_JSON_MARSHAL_ERROR, requestID, err).PrintfError()
			return nil, nil, http.StatusBadRequest, err
		}

		// формируем ответ
		header := _http.Header{}
		header[_http.HEADER_CONTENT_TYPE] = _http.HEADER_CONTENT_TYPE_JSON_UTF8
		header[_http.HEADER_CUSTOM_ERR_CODE] = _http.HEADER_CUSTOM_ERR_CODE_SUCCESS

		//_log.Debug("SUCCESS", requestID)

		return responseBuf, header, http.StatusOK, nil
	})

	//_log.Debug("SUCCESS ==================================================================================")
}

// calculateFactorial функция запуска расчета Factorial
func calculateFactorial(ctx context.Context, wpService *_wpservice.Service, requestID uint64, wpFactorialReqResp *WpFactorialReqResp, wpTipe string) (err error) {

	//var tic = time.Now()
	var tasks = make([]*_wp.Task, 0, len(wpFactorialReqResp.NumArray))

	// Подготовим список задач для запуска
	for i, value := range wpFactorialReqResp.NumArray {
		task := _wp.NewTask(ctx, "CalculateFactorial", nil, uint64(i), requestID, wpService.GetWPConfig().TaskTimeout, calculateFactorialFn, value)
		tasks = append(tasks, task)
	}

	// в конце обработки отправить task в кэш для повторного использования
	defer func() {
		for _, task := range tasks {
			task.Delete()
		}
	}()

	{ // Запускаем обработку
		if wpTipe == "bg" {
			// Запускаем обработку в общий background pool
			//_log.Debug("Start with global worker pool: requestID", requestID)
			err = wpService.RunTasksGroupWG(requestID, tasks, "Calculate - background")
		} else {
			// Запускаем обработку в локальный пул обработчиков
			var pool *_wp.Pool
			//_log.Debug("Start with local worker pool: calcId", requestID)
			pool, err = _wp.NewPool(ctx, requestID, "Calculate - online", wpService.GetWPConfig())
			if err == nil {
				err = pool.RunOnline(requestID, tasks, wpService.GetWPConfig().TaskTimeout)
			}
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
						return _err.NewTyped(_err.ERR_INCORRECT_TYPE_ERROR, _err.ERR_UNDEFINED_ID, "WpHandlerFactorial", "0 - uint", reflect.ValueOf(factorial).Type().String(), reflect.ValueOf(uint64(1)).Type().String()).PrintfError()
					}
				} else {
					return task.GetError()
				}
			}

			//wpFactorialReqResp.Duration = fmt.Sprintf("%s", time.Now().Sub(tic))
		} else {
			return err
		}
	} // Запускаем обработку

	return err
}

// calculateFactorialFn функция запуска расчета Factorial через worker pool
func calculateFactorialFn(ctx context.Context, data ...interface{}) (error, []interface{}) {
	var factVal uint64 = 1
	var cnt uint64 = 1
	var value uint64
	var ok bool

	if len(data) == 1 {

		// проверяем тип входных параметров
		if value, ok = data[0].(uint64); !ok {
			return _err.NewTyped(_err.ERR_INCORRECT_TYPE_ERROR, _err.ERR_UNDEFINED_ID, "CalculateFactorialFn", "0 - uint64", reflect.ValueOf(data[0]).Type().String(), reflect.ValueOf(uint64(1)).Type().String()).PrintfError(), nil
		} else {

			// Запускаем расчет
			for cnt = 1; cnt <= value; cnt++ {
				factVal *= cnt
				//time.Sleep(time.Millisecond * 20)
			}
		}

		return nil, []interface{}{factVal} // ошибки расчета транслируем на уровень выше
	}
	return _err.NewTyped(_err.ERR_INCORRECT_ARG_NUM_ERROR, _err.ERR_UNDEFINED_ID, data).PrintfError(), nil
}
