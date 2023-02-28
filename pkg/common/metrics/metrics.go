package metrics

import (
    "time"

    "github.com/golang/protobuf/proto"
    "github.com/povilasv/prommod"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/collectors"
    "github.com/prometheus/client_golang/prometheus/promauto"

    _err "github.com/romapres2010/goapp/pkg/common/error"
    _log "github.com/romapres2010/goapp/pkg/common/logger"
)

const (
    DEF_METRICS_NAMESPACE = "com"
    DEF_METRICS_SUBSYSTEM = "api"
)

// Config конфигурационные настройки
type Config struct {
    MetricsNamespace string `yaml:"metrics_namespace"`
    MetricsSubsystem string `yaml:"metrics_subsystem"`

    // Метрики DB
    CollectDBCountVec    bool `yaml:"collect_db_count_vec"`
    CollectDBDurationVec bool `yaml:"collect_db_duration_vec"`

    // Метрики HTTP request
    CollectHTTPRequestsCountVec      bool `yaml:"collect_http_requests_count_vec"`
    CollectHTTPErrorRequestsCountVec bool `yaml:"collect_http_error_requests_count_vec"`
    CollectHTTPRequestsDurationVec   bool `yaml:"collect_http_requests_duration_vec"`
    CollectHTTPActiveRequestsCount   bool `yaml:"collect_http_active_requests_count"`
    CollectHTTPRequestsDuration      bool `yaml:"collect_http_requests_duration"`

    // Метрики HTTP client call
    CollectHTTPClientCallCountVec    bool `yaml:"collect_http_client_call_count_vec"`
    CollectHTTPClientCallDurationVec bool `yaml:"collect_http_client_call_duration_vec"`

    // Метрики вычислений
    CollectCalcCountVec    bool `yaml:"collect_calc_count_vec"`
    CollectCalcDurationVec bool `yaml:"collect_calc_duration_vec"`

    // Метрики JSON
    CollectMarshalingDurationVec   bool `yaml:"collect_marshaling_duration_vec"`
    CollectUnMarshalingDurationVec bool `yaml:"collect_un_marshaling_duration_vec"`

    // Метрики WorkerPool
    CollectWPTaskQueueBufferLen bool `yaml:"collect_wp_task_queue_buffer_len"`
    CollectWPAddTaskWaitCount   bool `yaml:"collect_wp_add_task_wait_count"`
    CollectWPWorkerProcessCount bool `yaml:"collect_wp_worker_process_count"`

    // Метрики WorkerPool
    CollectWPTaskQueueBufferLenVec  bool `yaml:"collect_wp_task_queue_buffer_len_vec"`
    CollectWPAddTaskWaitCountVec    bool `yaml:"collect_wp_add_task_wait_count_vec"`
    CollectWPWorkerProcessCountVec  bool `yaml:"collect_wp_worker_process_count_vec"`
    CollectWPTaskProcessDurationVec bool `yaml:"collect_wp_task_process_duration_ms_by_name"`
}

type Metrics struct {
    Cfg      *Config
    Registry *prometheus.Registry

    // Метрики DB
    DBCountVec    *prometheus.CounterVec
    DBDurationVec *prometheus.HistogramVec

    // Метрики HTTP request
    HTTPRequestsCountVec      *prometheus.CounterVec
    HTTPErrorRequestsCountVec *prometheus.CounterVec
    HTTPRequestsDurationVec   *prometheus.HistogramVec
    HTTPActiveRequestsCount   prometheus.Gauge
    HTTPRequestsDuration      prometheus.Histogram

    // Метрики HTTP client call
    HTTPClientCallCountVec    *prometheus.CounterVec
    HTTPClientCallDurationVec *prometheus.HistogramVec

    // Метрики вычислений
    CalcCountVec    *prometheus.CounterVec
    CalcDurationVec *prometheus.HistogramVec

    // Метрики JSON
    MarshalingDurationVec   *prometheus.HistogramVec
    UnMarshalingDurationVec *prometheus.HistogramVec

    // Метрики WorkerPoolVec
    WPTaskQueueBufferLenVec  *prometheus.GaugeVec
    WPAddTaskWaitCountVec    *prometheus.GaugeVec
    WPWorkerProcessCountVec  *prometheus.GaugeVec
    WPTaskProcessDurationVec *prometheus.HistogramVec
}

func New(cfg *Config) (*Metrics, error) {
    _log.Info("Creating new metrics service")

    { // входные проверки
        if cfg == nil {
            return nil, _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, _err.ERR_UNDEFINED_ID, "if cfg == nil {}").PrintfError()
        }
    } // входные проверки

    if cfg.MetricsNamespace == "" {
        cfg.MetricsNamespace = DEF_METRICS_NAMESPACE
    }

    if cfg.MetricsSubsystem == "" {
        cfg.MetricsSubsystem = DEF_METRICS_SUBSYSTEM
    }

    var metrics = Metrics{}

    metrics.Cfg = cfg
    metrics.Registry = prometheus.NewRegistry()

    // Add Go module build info.
    metrics.Registry.MustRegister(collectors.NewBuildInfoCollector())
    //metrics.Registry.MustRegister(collectors.NewGoCollector(
    //    collectors.WithGoCollections(collectors.GoRuntimeMemStatsCollection | collectors.GoRuntimeMetricsCollection),
    //))
    metrics.Registry.MustRegister(collectors.NewGoCollector(
        collectors.WithGoCollectorRuntimeMetrics(),
    ))

    // вывод в метрики всех зависимых модулей
    err := metrics.Registry.Register(prommod.NewCollector("app_name"))
    if err != nil {
        return nil, err
    }

    _log.Info("Dependency: ", "\n"+prommod.Print("app_name"))

    { // Метрики DB
        metrics.DBCountVec = promauto.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: cfg.MetricsNamespace,
                Subsystem: cfg.MetricsSubsystem,
                Name:      "db_total",
                Help:      "The total number of processed DB by sql statement",
            },
            []string{"SQL"},
        )
        metrics.DBDurationVec = promauto.NewHistogramVec(
            prometheus.HistogramOpts{
                Namespace: cfg.MetricsNamespace,
                Subsystem: cfg.MetricsSubsystem,
                Name:      "db_duration_ms",
                Help:      "The duration histogram of DB operation in ms by sql statement",
                Buckets:   []float64{0.01, 0.05, 0.10, 0.50, 1.00, 5.00, 10.00},
            },
            []string{"SQL"},
        )
        metrics.Registry.MustRegister(metrics.DBCountVec)
        metrics.Registry.MustRegister(metrics.DBDurationVec)
    } // Метрики DB

    { // Метрики HTTP
        metrics.HTTPRequestsCountVec = prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: cfg.MetricsNamespace,
                Subsystem: cfg.MetricsSubsystem,
                Name:      "http_requests_total_by_resource",
                Help:      "How many HTTP requests processed, partitioned by resource",
            },
            []string{"resource", "method"},
        )

        metrics.HTTPErrorRequestsCountVec = prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: cfg.MetricsNamespace,
                Subsystem: cfg.MetricsSubsystem,
                Name:      "http_requests_error_total_by_resource",
                Help:      "How many HTTP requests was ERRORED, partitioned by resource",
            },
            []string{"resource", "method"},
        )

        metrics.HTTPRequestsDurationVec = prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Namespace: cfg.MetricsNamespace,
                Subsystem: cfg.MetricsSubsystem,
                Name:      "http_request_duration_ms_by_resource",
                Help:      "The duration histogram of HTTP requests in ms by resource",
                Buckets:   []float64{0.1, 5, 10, 50, 100, 500, 1000},
            },
            []string{"resource", "method"},
        )

        metrics.HTTPActiveRequestsCount = promauto.NewGauge(
            prometheus.GaugeOpts{
                Namespace: cfg.MetricsNamespace,
                Subsystem: cfg.MetricsSubsystem,
                Name:      "http_active_requests_count",
                Help:      "The total number of active HTTP requests",
            })

        metrics.HTTPRequestsDuration = promauto.NewHistogram(
            prometheus.HistogramOpts{
                Namespace: cfg.MetricsNamespace,
                Subsystem: cfg.MetricsSubsystem,
                Name:      "http_request_duration_ms",
                Help:      "The duration histogram of HTTP requests in ms",
                Buckets:   []float64{0.1, 5, 10, 50, 100, 500, 1000},
            })

        metrics.Registry.MustRegister(metrics.HTTPRequestsCountVec)
        metrics.Registry.MustRegister(metrics.HTTPErrorRequestsCountVec)
        metrics.Registry.MustRegister(metrics.HTTPRequestsDurationVec)
        metrics.Registry.MustRegister(metrics.HTTPActiveRequestsCount)
        metrics.Registry.MustRegister(metrics.HTTPRequestsDuration)
    } // Метрики HTTP

    { // Метрики HTTP client call
        metrics.HTTPClientCallCountVec = prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: cfg.MetricsNamespace,
                Subsystem: cfg.MetricsSubsystem,
                Name:      "http_client_call_total_by_resource",
                Help:      "How many HTTP client call processed, partitioned by resource",
            },
            []string{"resource", "method"},
        )

        metrics.HTTPClientCallDurationVec = prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Namespace: cfg.MetricsNamespace,
                Subsystem: cfg.MetricsSubsystem,
                Name:      "http_client_call_duration_ms_by_resource",
                Help:      "The duration histogram of HTTP client call in ms by resource",
                Buckets:   []float64{0.1, 5, 10, 50, 100, 500, 1000},
            },
            []string{"resource", "method"},
        )

        metrics.Registry.MustRegister(metrics.HTTPClientCallCountVec)
        metrics.Registry.MustRegister(metrics.HTTPClientCallDurationVec)
    } // Метрики HTTP client call

    { // Метрики вычислений
        metrics.CalcCountVec = prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: cfg.MetricsNamespace,
                Subsystem: cfg.MetricsSubsystem,
                Name:      "calc_total_by_type",
                Help:      "How many calculation processed, partitioned by type",
            },
            []string{"type"},
        )

        metrics.CalcDurationVec = prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Namespace: cfg.MetricsNamespace,
                Subsystem: cfg.MetricsSubsystem,
                Name:      "calc_duration_by_type",
                Help:      "The duration histogram of calculation in ms by type",
                Buckets:   []float64{0.1, 5, 10, 50, 100, 500, 1000},
            },
            []string{"type"},
        )

        metrics.Registry.MustRegister(metrics.CalcCountVec)
        metrics.Registry.MustRegister(metrics.CalcDurationVec)
    } // Метрики вычислений

    { // Метрики JSON
        metrics.MarshalingDurationVec = prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Namespace: cfg.MetricsNamespace,
                Subsystem: cfg.MetricsSubsystem,
                Name:      "json_marshaling_duration_by_type",
                Help:      "The duration histogram of JSON marshaling in ms by type",
                Buckets:   []float64{0.01, 0.05, 0.10, 0.50, 1.00, 5.00, 10.00},
            },
            []string{"type"},
        )

        metrics.UnMarshalingDurationVec = prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Namespace: cfg.MetricsNamespace,
                Subsystem: cfg.MetricsSubsystem,
                Name:      "json_unmarshaling_duration_by_type",
                Help:      "The duration histogram of JSON unmarshaling in ms by type",
                Buckets:   []float64{0.01, 0.05, 0.10, 0.50, 1.00, 5.00, 10.00},
            },
            []string{"type"},
        )

        metrics.Registry.MustRegister(metrics.MarshalingDurationVec)
        metrics.Registry.MustRegister(metrics.UnMarshalingDurationVec)
    } // Метрики JSON

    { // Метрики WorkerPool
        metrics.WPTaskQueueBufferLenVec = promauto.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: cfg.MetricsNamespace,
                Subsystem: cfg.MetricsSubsystem,
                Name:      "wp_task_queue_buffer_len_vec",
                Help:      "The len of the worker pool buffer",
            },
            []string{"type"},
        )

        metrics.WPAddTaskWaitCountVec = promauto.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: cfg.MetricsNamespace,
                Subsystem: cfg.MetricsSubsystem,
                Name:      "wp_add_task_wait_count_vec",
                Help:      "The number of the task waiting to add to worker pool queue",
            },
            []string{"type"},
        )

        metrics.WPWorkerProcessCountVec = promauto.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: cfg.MetricsNamespace,
                Subsystem: cfg.MetricsSubsystem,
                Name:      "wp_worker_process_count_vec",
                Help:      "The number of the working worker",
            },
            []string{"type"},
        )

        metrics.WPTaskProcessDurationVec = prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Namespace: cfg.MetricsNamespace,
                Subsystem: cfg.MetricsSubsystem,
                Name:      "wp_task_process_duration_ms_by_name",
                Help:      "The duration histogram of worker pool task process in ms by name",
                Buckets:   []float64{0.1, 5, 10, 50, 100, 500, 1000},
            },
            []string{"type", "name"},
        )

        metrics.Registry.MustRegister(metrics.WPTaskQueueBufferLenVec)
        metrics.Registry.MustRegister(metrics.WPAddTaskWaitCountVec)
        metrics.Registry.MustRegister(metrics.WPWorkerProcessCountVec)
        metrics.Registry.MustRegister(metrics.WPTaskProcessDurationVec)
    } // Метрики WorkerPool

    return &metrics, nil
}

func (mt *Metrics) PrintMetricsToLog() {

    metricFamilies, err := mt.Registry.Gather()
    if err != nil {
        _ = _err.WithCauseTyped(_err.ERR_COMMON_ERROR, _err.ERR_UNDEFINED_ID, err, err.Error()).PrintfError()
    }
    for _, metric := range metricFamilies {
        _log.Info("Metrics: ", "\n"+proto.MarshalTextString(metric))
    }
}

var GlobalMetrics *Metrics

// Метрики DB
func IncDBCountVec(sql string) {
    if GlobalMetrics.Cfg.CollectDBCountVec {
        GlobalMetrics.DBCountVec.WithLabelValues(sql).Inc()
    }
}
func AddDBDurationVec(sql string, duration time.Duration) {
    if GlobalMetrics.Cfg.CollectDBDurationVec {
        GlobalMetrics.DBDurationVec.WithLabelValues(sql).Observe(float64(duration.Milliseconds()))
    }
}

// Метрики HTTP
func IncHTTPRequestsCountVec(resource string, method string) {
    if GlobalMetrics.Cfg.CollectHTTPRequestsCountVec {
        GlobalMetrics.HTTPRequestsCountVec.WithLabelValues(resource, method).Inc()
    }
}
func IncHTTPErrorRequestsCountVec(resource string, method string) {
    if GlobalMetrics.Cfg.CollectHTTPErrorRequestsCountVec {
        GlobalMetrics.HTTPErrorRequestsCountVec.WithLabelValues(resource, method).Inc()
    }
}
func IncHTTPRequestsDurationVec(resource string, method string, state string, duration time.Duration) {
    if GlobalMetrics.Cfg.CollectHTTPRequestsDurationVec {
        GlobalMetrics.HTTPRequestsDurationVec.WithLabelValues(resource, method+": "+state).Observe(float64(duration.Milliseconds()))
    }
}
func IncHTTPActiveRequestsCount() {
    if GlobalMetrics.Cfg.CollectHTTPActiveRequestsCount {
        GlobalMetrics.HTTPActiveRequestsCount.Inc()
    }
}
func DecHTTPActiveRequestsCount() {
    if GlobalMetrics.Cfg.CollectHTTPActiveRequestsCount {
        GlobalMetrics.HTTPActiveRequestsCount.Dec()
    }
}
func AddHTTPRequestsDuration(duration time.Duration) {
    if GlobalMetrics.Cfg.CollectHTTPRequestsDuration {
        GlobalMetrics.HTTPRequestsDuration.Observe(float64(duration.Milliseconds()))
    }
}

// Метрики HTTP client call
func IncHTTPClientCallCountVec(resource string, method string) {
    if GlobalMetrics.Cfg.CollectHTTPClientCallCountVec {
        GlobalMetrics.HTTPClientCallCountVec.WithLabelValues(resource, method).Inc()
    }
}
func IncHTTPClientCallDurationVec(resource string, method string, duration time.Duration) {
    if GlobalMetrics.Cfg.CollectHTTPClientCallDurationVec {
        GlobalMetrics.HTTPClientCallDurationVec.WithLabelValues(resource, method).Observe(float64(duration.Milliseconds()))
    }
}

// Метрики вычислений
func IncCalcCountVec(label string) {
    if GlobalMetrics.Cfg.CollectCalcCountVec {
        GlobalMetrics.CalcCountVec.WithLabelValues(label).Inc()
    }
}
func IncCalcDurationVec(label string, duration time.Duration) {
    if GlobalMetrics.Cfg.CollectCalcDurationVec {
        GlobalMetrics.CalcDurationVec.WithLabelValues(label).Observe(float64(duration.Milliseconds()))
    }
}

// Метрики JSON
func IncMarshalingDurationVec(label string, duration time.Duration) {
    if GlobalMetrics.Cfg.CollectMarshalingDurationVec {
        GlobalMetrics.MarshalingDurationVec.WithLabelValues(label).Observe(float64(duration.Milliseconds()))
    }
}
func IncUnMarshalingDurationVec(label string, duration time.Duration) {
    if GlobalMetrics.Cfg.CollectUnMarshalingDurationVec {
        GlobalMetrics.UnMarshalingDurationVec.WithLabelValues(label).Observe(float64(duration.Milliseconds()))
    }
}

// Метрики WorkerPoolVec
func IncWPTaskQueueBufferLenVec(wpType string) {
    if GlobalMetrics.Cfg.CollectWPTaskQueueBufferLenVec {
        GlobalMetrics.WPTaskQueueBufferLenVec.WithLabelValues(wpType).Inc()
    }
}
func DecWPTaskQueueBufferLenVec(wpType string) {
    if GlobalMetrics.Cfg.CollectWPTaskQueueBufferLenVec {
        GlobalMetrics.WPTaskQueueBufferLenVec.WithLabelValues(wpType).Dec()
    }
}
func SetWPTaskQueueBufferLenVec(wpType string, len float64) {
    if GlobalMetrics.Cfg.CollectWPTaskQueueBufferLenVec {
        GlobalMetrics.WPTaskQueueBufferLenVec.WithLabelValues(wpType).Set(len)
    }
}
func IncWPAddTaskWaitCountVec(wpType string) {
    if GlobalMetrics.Cfg.CollectWPAddTaskWaitCountVec {
        GlobalMetrics.WPAddTaskWaitCountVec.WithLabelValues(wpType).Inc()
    }
}
func DecWPAddTaskWaitCountVec(wpType string) {
    if GlobalMetrics.Cfg.CollectWPAddTaskWaitCountVec {
        GlobalMetrics.WPAddTaskWaitCountVec.WithLabelValues(wpType).Dec()
    }
}
func IncWPWorkerProcessCountVec(wpType string) {
    if GlobalMetrics.Cfg.CollectWPWorkerProcessCountVec {
        GlobalMetrics.WPWorkerProcessCountVec.WithLabelValues(wpType).Inc()
    }
}
func DecWPWorkerProcessCountVec(wpType string) {
    if GlobalMetrics.Cfg.CollectWPWorkerProcessCountVec {
        GlobalMetrics.WPWorkerProcessCountVec.WithLabelValues(wpType).Dec()
    }
}
func IncWPTaskProcessDurationVec(wpType string, name string, duration time.Duration) {
    if GlobalMetrics.Cfg.CollectWPTaskProcessDurationVec {
        GlobalMetrics.WPTaskProcessDurationVec.WithLabelValues(wpType, name).Observe(float64(duration.Milliseconds()))
    }
}

func PrintGlobalMetricsToLog() {
    GlobalMetrics.PrintMetricsToLog()
}

func InitGlobalMetrics(cfg *Config) {
    GlobalMetrics, _ = New(cfg)
}
