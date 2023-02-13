package logger

import (
    "fmt"
    "os"
    "runtime"
    "strconv"
    "strings"
    "time"

    pkgerr "github.com/pkg/errors"
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
    "gopkg.in/natefinch/lumberjack.v2"
)

const LEVEL_DEBUG = "DEBUG"
const LEVEL_INFO = "INFO"
const LEVEL_WARNING = "WARNING"
const LEVEL_ERROR = "ERROR"
const LEVEL_DPANIC = "DPANIC"
const LEVEL_PANIC = "PANIC"
const LEVEL_FATAL = "FATAL"

// Config конфигурация сервиса логирования
type Config struct {
    Enable         bool   `json:"enable" yaml:"enable"`                   // состояние логирования 'true', 'false'
    GlobalLevel    string `json:"global_level" yaml:"global_level"`       // debug, info, warn, error, dpanic, panic, fatal - все логгеры ниже этого уровня будут отключены
    GlobalFilename string `json:"global_filename" yaml:"global_filename"` // глобальное имя файл для логирования

    ZapConfig ZapConfig `json:"zap" yaml:"zap"`
}

// FileConfig
type FileConfig struct {
    Filename   string `json:"filename" yaml:"filename"`       // имя файл для логирования
    MaxSize    int    `json:"max_size" yaml:"max_size"`       // максимальный размер лог файла
    MaxAge     int    `json:"max_age" yaml:"max_age"`         // время хранения истории лог файлов в днях
    MaxBackups int    `json:"max_backups" yaml:"max_backups"` // максимальное количество архивных логов
    LocalTime  bool   `json:"local_time" yaml:"local_time"`   // использовать локальное время в имени архивных лог файлов
    Compress   bool   `json:"compress" yaml:"compress"`       // сжимать архивные лог файлы в zip архив
}

// StdConfig
type StdConfig struct {
    Enable      bool   `json:"enable" yaml:"enable"`               // состояние логирования 'true', 'false'
    Level       string `json:"log_level" yaml:"log_level"`         // уровень логирования 'DEBUG', 'INFO', 'ERROR'
    LogToFile   bool   `json:"log_to_file" yaml:"log_to_file"`     // логировать в файл 'true', 'false'
    LogToStderr bool   `json:"log_to_stderr" yaml:"log_to_stderr"` // логировать в stderr 'true', 'false'
    LogFlag     int    `json:"log_flag" yaml:"log_flag"`           // флаги настройки вывода лога
    LogPrefix   string `json:"log_prefix" yaml:"log_prefix"`       // префикс строк в лог файл

    FileConfig *FileConfig `json:"file" yaml:"file"` // конфигурация для логирования в файл, если имя файла пустое, то используется глобальное
}

// ZapConfig
type ZapConfig struct {
    Enable            bool   `json:"enable" yaml:"enable"`                                   // состояние логирования 'true', 'false'
    DisableCaller     bool   `json:"disable_caller,omitempty" yaml:"disable_caller"`         // запретить вывод в лог информации о caller
    DisableStacktrace bool   `json:"disable_stacktrace,omitempty" yaml:"disable_stacktrace"` // запретить вывод в stacktrace
    StacktraceLevel   string `json:"stacktrace_level" yaml:"stacktrace_level"`               // для какого уровня выводить stacktrace debug, info, warn, error, dpanic, panic, fatal
    Development       bool   `json:"development" yaml:"development"`                         // режим разработки для уровня dpanic

    ZapCoreConfig []ZapCoreConfig `json:"core,omitempty" yaml:"core"`

    Sampling *zap.SamplingConfig `json:"sampling,omitempty" yaml:"sampling"`
}

// ZapCoreConfig
type ZapCoreConfig struct {
    Enable        bool             `json:"enable" yaml:"enable"`                 // состояние логирования 'true', 'false'
    MinLevel      string           `json:"min_level" yaml:"min_level"`           // минимальный уровень логирования debug, info, warn, error, dpanic, panic, fatal
    MaxLevel      string           `json:"max_level" yaml:"max_level"`           // максимальный уровень логирования debug, info, warn, error, dpanic, panic, fatal
    LogTo         string           `json:"log_to" yaml:"log_to"`                 // логировать в 'file', 'stderr', 'stdout', 'url', 'lumberjack'
    LoggingPath   string           `json:"logging_path" yaml:"logging_path"`     // путь для логирования
    Encoding      string           `json:"encoding,omitempty" yaml:"encoding"`   // формат вывода 'console', 'json'
    FileConfig    *FileConfig      `json:"file" yaml:"file"`                     // конфигурация для логирования в файл, если имя файла пустое, то используется глобальное
    EncoderConfig ZapEncoderConfig `json:"encoder_config" yaml:"encoder_config"` // конфигурация Encoder
}

// ZapEncoderConfig
type ZapEncoderConfig struct {
    MessageKey       string `json:"message_key,omitempty" yaml:"message_key"`
    LevelKey         string `json:"level_key,omitempty" yaml:"level_key"`
    TimeKey          string `json:"time_key,omitempty" yaml:"time_key"`
    NameKey          string `json:"name_key,omitempty" yaml:"name_key"`
    CallerKey        string `json:"caller_key,omitempty" yaml:"caller_key"`
    FunctionKey      string `json:"function_key,omitempty" yaml:"function_key"`
    StacktraceKey    string `json:"stacktrace_key,omitempty" yaml:"stacktrace_key"`
    SkipLineEnding   bool   `json:"skip_line_ending,omitempty" yaml:"skip_line_ending"`
    LineEnding       string `json:"line_ending,omitempty" yaml:"line_ending"`
    EncodeLevel      string `json:"encode_level,omitempty" yaml:"encode_level"`             // capital, capitalColor, color, lower
    EncodeTime       string `json:"encode_time,omitempty" yaml:"encode_time"`               // rfc3339nano, rfc3339, iso8601, millis, nanos
    EncodeTimeCustom string `json:"encode_time_custom,omitempty" yaml:"encode_time_custom"` // 2006-01-02 15:04:05.000000
    EncodeDuration   string `json:"encode_duration,omitempty" yaml:"encode_duration"`       // string, nanos, ms
    EncodeCaller     string `json:"encode_caller,omitempty" yaml:"encode_caller"`           // full, short
    ConsoleSeparator string `json:"console_separator,omitempty" yaml:"console_separator"`
}

var zapEncoderDefConfig = zapcore.EncoderConfig{
    TimeKey:          "ts",
    LevelKey:         "level",
    NameKey:          "logger",
    CallerKey:        "caller",
    FunctionKey:      zapcore.OmitKey,
    MessageKey:       "msg",
    StacktraceKey:    "stacktrace",
    LineEnding:       zapcore.DefaultLineEnding,
    EncodeLevel:      zapcore.LowercaseLevelEncoder,
    EncodeTime:       zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000000"),
    EncodeDuration:   zapcore.MillisDurationEncoder,
    EncodeCaller:     zapcore.ShortCallerEncoder,
    ConsoleSeparator: "    ",
}

var zapDefConfig = zap.Config{
    Level:       zap.NewAtomicLevelAt(zapcore.InfoLevel),
    Development: true,
    Sampling: &zap.SamplingConfig{
        Initial:    100,
        Thereafter: 100,
    },
    Encoding:         "console",
    EncoderConfig:    zapEncoderDefConfig,
    OutputPaths:      []string{"stdout"},
    ErrorOutputPaths: []string{"stderr"},
}

// gLogDefCfg - дефолтные настройки Logger
var gLogDefCfg = Config{
    Enable:         true,
    GlobalLevel:    LEVEL_INFO,
    GlobalFilename: "./log/default.log",
    ZapConfig: ZapConfig{
        Enable: true,
    },
}

// gLogCfg - настройки глобального Logger
var gLogCfg = gLogDefCfg

// gZapLogger - глобальный zap Logger
var gZapLogger *zap.Logger

// gZapDefLogger - глобальный default zap Logger
var gZapDefLogger *zap.Logger

var gZapGlobalLevel zapcore.Level

// gClosers - массив для закрытия zap Logger
var gClosers []func()

func gClose() {
    Info("Logger - close all")
    for _, f := range gClosers {
        f()
    }
}

// https://stackoverflow.com/questions/54395407/zap-logging-with-1-customized-config-and-2-lumberjack
// Sync implements zap.Sink. The remaining methods are implemented by the embedded *lumberjack.Logger.
type lumberjackSink struct{ *lumberjack.Logger }

func (lumberjackSink) Sync() error { return nil }

func init() {
    // Дефолтная настройка ZapLogger
    gZapDefLogger = zap.Must(zapDefConfig.Build(zap.WithCaller(true), zap.AddCallerSkip(2), zap.AddStacktrace(zapcore.ErrorLevel)))
    gZapLogger = gZapDefLogger

    gZapGlobalLevel = zapcore.InfoLevel
}

func GlobalZapLogger() *zap.Logger {
    return gZapLogger
}

func GlobalConfig() Config {
    return gLogCfg
}

// SetGlobalConfig - apply new config
func SetGlobalConfig(cfg Config) (err error) {

    _ = CloseLogger()

    gLogCfg = cfg

    zapAtomicLevel, err := zap.ParseAtomicLevel(cfg.GlobalLevel)
    if err != nil {
        return err
    }
    gZapGlobalLevel = zapAtomicLevel.Level()

    if cfg.Enable {
        gZapLogger, err = newZapLogger(cfg.ZapConfig, cfg.GlobalFilename)
        if err != nil {
            return err
        }
    }

    Info("Logger settings applied: cfg", cfg)
    return err
}

// NewFileLogger - создать новый логгер в файл
func NewFileLogger(cfg FileConfig, globalFileName string) *lumberjack.Logger {

    var fileLogger = &lumberjack.Logger{}
    var fileName = globalFileName + cfg.Filename

    Info("File logger - log to file: FileName", fileName)

    { // Скопируем конфигурацию
        fileLogger.MaxSize = cfg.MaxSize
        fileLogger.MaxAge = cfg.MaxAge
        fileLogger.MaxBackups = cfg.MaxBackups
        fileLogger.LocalTime = cfg.LocalTime
        fileLogger.Compress = cfg.Compress

        if fileName != "" {
            fileLogger.Filename = fileName
        } else {
            Info("File logger - log to default file: FileName", gLogDefCfg.GlobalFilename)
            fileLogger.Filename = gLogDefCfg.GlobalFilename
        }
    } // Скопируем конфигурацию

    // добавим для закрытия при выходе
    _close := func() {
        err := fileLogger.Close()
        if err != nil {
            Error(err.Error())
        }
    }
    gClosers = append(gClosers, _close)

    return fileLogger
}

// newZapLogger - создать zap.Logger на основании YAML
func newZapLogger(cfg ZapConfig, globalFileName string) (*zap.Logger, error) {

    if cfg.Enable {
        var err error
        var options []zap.Option
        var cores []zapcore.Core
        var writeSyncer zapcore.WriteSyncer

        Info("Setup zap logger")

        if !cfg.DisableCaller {
            options = append(options, zap.WithCaller(true))
            options = append(options, zap.AddCallerSkip(2))
        }

        if !cfg.Development {
            options = append(options, zap.Development())
        }

        if !cfg.DisableStacktrace {
            level, err := zap.ParseAtomicLevel(cfg.StacktraceLevel)
            if err != nil {
                return nil, err
            }
            options = append(options, zap.AddStacktrace(level))
        }

        for _, coreConfig := range cfg.ZapCoreConfig {

            if coreConfig.Enable {
                Info("Zap logger - add core: minLevel, maxLevel, log_to, encoding", coreConfig.MinLevel, coreConfig.MaxLevel, coreConfig.LogTo, coreConfig.Encoding)

                var _close func()
                switch coreConfig.LogTo {
                case "lumberjack":
                    if coreConfig.FileConfig != nil {
                        writeSyncer = zapcore.AddSync(NewFileLogger(*coreConfig.FileConfig, globalFileName))
                    } else {
                        gClose()
                        return nil, pkgerr.New("Missing FileConfig in CoreConfig")
                    }
                case "file":
                    fileName := globalFileName + coreConfig.LoggingPath
                    writeSyncer, _close, err = zap.Open(fileName)
                    if err != nil {
                        gClose()
                        return nil, err
                    }
                    // добавим для закрытия при выходе
                    gClosers = append(gClosers, _close)
                case "stderr":
                    writeSyncer = zapcore.Lock(WrappedWriteSyncer{os.Stderr})
                case "stdout":
                    writeSyncer = zapcore.Lock(WrappedWriteSyncer{os.Stdout})
                case "url":
                    writeSyncer, _, err = zap.Open(coreConfig.LoggingPath)
                    if err != nil {
                        gClose()
                        return nil, err
                    }
                default:
                    writeSyncer = zapcore.Lock(WrappedWriteSyncer{os.Stdout})
                }

                core, err := newZapCore(coreConfig, writeSyncer)
                if err != nil {
                    gClose()
                    return nil, err
                }
                cores = append(cores, core)
            } else {
                Info("Zap logger - core DISABLE: log_to, encoding", coreConfig.LogTo, coreConfig.Encoding)
            }
        }

        return zap.New(zapcore.NewTee(cores...), options...), nil
    } else {
        Info("Zap logger - DISABLE")
        return nil, nil
    }
}

// newZapCore - создать zap.Core на основании YAML
func newZapCore(cfg ZapCoreConfig, writeSyncer zapcore.WriteSyncer) (zapcore.Core, error) {
    var minLevelSrt = cfg.MinLevel
    var maxLevelSrt = cfg.MaxLevel

    if minLevelSrt == "" {
        minLevelSrt = "DEBUG"
    }

    minAtomicLevel, err := zap.ParseAtomicLevel(minLevelSrt)
    if err != nil {
        return nil, err
    }
    minZapLevel := minAtomicLevel.Level()

    if maxLevelSrt == "" {
        maxLevelSrt = "FATAL"
    }

    maxAtomicLevel, err := zap.ParseAtomicLevel(maxLevelSrt)
    if err != nil {
        return nil, err
    }
    maxZapLevel := maxAtomicLevel.Level()

    levelEnabler := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
        return lvl >= minZapLevel && lvl <= maxZapLevel
    })

    var encoder zapcore.Encoder
    switch cfg.Encoding {
    case "console":
        encoder = zapcore.NewConsoleEncoder(newZapEncoderConfig(cfg.EncoderConfig))
    case "json":
        encoder = zapcore.NewJSONEncoder(newZapEncoderConfig(cfg.EncoderConfig))
    default:
        encoder = zapcore.NewConsoleEncoder(newZapEncoderConfig(cfg.EncoderConfig))
    }

    return zapcore.NewCore(encoder, writeSyncer, levelEnabler), err
}

// newZapEncoderConfig - создать конфиг на основании YAML
func newZapEncoderConfig(cfg ZapEncoderConfig) zapcore.EncoderConfig {
    encoderConfig := zapEncoderDefConfig

    encoderConfig.MessageKey = cfg.MessageKey
    encoderConfig.LevelKey = cfg.LevelKey
    encoderConfig.TimeKey = cfg.TimeKey
    encoderConfig.NameKey = cfg.NameKey
    encoderConfig.CallerKey = cfg.CallerKey
    encoderConfig.FunctionKey = cfg.FunctionKey
    encoderConfig.StacktraceKey = cfg.StacktraceKey
    encoderConfig.SkipLineEnding = cfg.SkipLineEnding
    encoderConfig.LineEnding = cfg.LineEnding

    EncodeDuration := new(zapcore.DurationEncoder)
    _ = EncodeDuration.UnmarshalText([]byte(cfg.EncodeDuration))
    encoderConfig.EncodeDuration = *EncodeDuration

    EncodeLevel := new(zapcore.LevelEncoder)
    _ = EncodeLevel.UnmarshalText([]byte(cfg.EncodeLevel))
    encoderConfig.EncodeLevel = *EncodeLevel

    if cfg.EncodeTimeCustom == "" {
        EncodeTime := new(zapcore.TimeEncoder)
        _ = EncodeTime.UnmarshalText([]byte(cfg.EncodeTime))
        encoderConfig.EncodeTime = *EncodeTime
    } else {
        encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(cfg.EncodeTimeCustom)
    }

    EncodeCaller := new(zapcore.CallerEncoder)
    _ = EncodeCaller.UnmarshalText([]byte(cfg.EncodeCaller))
    encoderConfig.EncodeCaller = *EncodeCaller

    return encoderConfig
}

// WrappedWriteSyncer is a helper struct implementing zapcore.WriteSyncer to
// wrap a standard os.Stdout handle, giving control over the WriteSyncer's
// Sync() function. Sync() results in an error on Windows in combination with
// os.Stdout ("sync /dev/stdout: The handle is invalid."). WrappedWriteSyncer
// simply does nothing when Sync() is called by Zap.
// https://github.com/uber-go/zap/issues/991
type WrappedWriteSyncer struct{ file *os.File }

func (mws WrappedWriteSyncer) Write(p []byte) (n int, err error) { return mws.file.Write(p) }
func (mws WrappedWriteSyncer) Sync() error                       { return nil }

// CloseLogger - закрыть все Logger и создать дефолтный
func CloseLogger() error {
    if GlobalZapLogger() != nil {
        err := GlobalZapLogger().Sync()
        if err != nil {
            Error(err.Error())
            //gClose()
            //return err
        }
    }

    gClose()
    gClosers = make([]func(), 0, 0) // пересоздать массив

    // Дефолтная настройка ZapLogger
    gZapLogger = gZapDefLogger

    gZapGlobalLevel = zapcore.InfoLevel

    return nil
}

// SetStdFilterLevel set log level
func SetStdFilterLevel(lev string) {
    Info("Set log level", lev)
}

// Info print message in Info level
func Info(mes string, args ...interface{}) {
    Log(LEVEL_INFO, 0, mes, args...)
}

// Debug print message in Debug level
func Debug(mes string, args ...interface{}) {
    Log(LEVEL_DEBUG, 0, mes, args...)
}

// ErrorAsInfo print error in Info level
func ErrorAsInfo(err error, args ...interface{}) {
    Log(LEVEL_INFO, 0, err.Error(), args...)
}

// Error print message in Error level
func Error(mes string, args ...interface{}) {
    Log(LEVEL_ERROR, 0, mes, args...)
}

// parseZapLevel - сформировать zapcore.Level
func parseZapLevel(level string) zapcore.Level {
    var zapLvl zapcore.Level
    switch level {
    case LEVEL_DEBUG:
        zapLvl = zapcore.DebugLevel
    case LEVEL_INFO:
        zapLvl = zapcore.InfoLevel
    case LEVEL_WARNING:
        zapLvl = zapcore.WarnLevel
    case LEVEL_ERROR:
        zapLvl = zapcore.ErrorLevel
    case LEVEL_DPANIC:
        zapLvl = zapcore.DPanicLevel
    case LEVEL_PANIC:
        zapLvl = zapcore.PanicLevel
    case LEVEL_FATAL:
        zapLvl = zapcore.FatalLevel
    default:
        zapLvl = zapcore.InfoLevel
    }
    return zapLvl
}

// Check - проверить, что уровень выше глобального и есть подходящие логеры
func Check(level string) bool {
    if gLogCfg.ZapConfig.Enable && gZapLogger != nil {
        zapLvl := parseZapLevel(level)
        // Если уровень сообщения выше глобального и есть подходящие логеры
        if zapLvl >= gZapGlobalLevel && zapLvl >= gZapLogger.Level() {
            return true
        }
    }
    return false
}

// Log print message
func Log(level string, depth int, mes string, args ...interface{}) {

    if gLogCfg.ZapConfig.Enable && gZapLogger != nil {

        // Если уровень сообщения выше глобального и есть подходящие логеры
        if Check(level) {
            zapLvl := parseZapLevel(level)
            if len(args) == 0 {
                gZapLogger.Log(zapLvl, strings.Join([]string{GetCallerShort(depth + 3), " - ", mes}, ""))
            } else {
                argsStr := getArgsString(args...) // get formatted string with arguments
                gZapLogger.Log(zapLvl, strings.Join([]string{GetCallerShort(depth + 3), " - ", mes, "[", argsStr, "]"}, ""))

            }
        }
    }
}

// getArgsString return formatted string with arguments
func getArgsString(args ...interface{}) (argsStr string) {
    for _, arg := range args {
        if arg != nil {
            argsStr = argsStr + fmt.Sprintf("'%v', ", arg)
        }
    }
    argsStr = strings.TrimRight(argsStr, ", ")
    return
}

// GetCaller returns a Valuer that returns a file and line from a specified depth in the callstack.
func GetCaller(depth int) string {
    pc := make([]uintptr, 15)
    n := runtime.Callers(depth+1, pc)
    frame, _ := runtime.CallersFrames(pc[:n]).Next()
    idxFile := strings.LastIndexByte(frame.File, '/')
    idx := strings.LastIndexByte(frame.Function, '/')
    idxName := strings.IndexByte(frame.Function[idx+1:], '.') + idx + 1

    return frame.File[idxFile+1:] + ":[" + strconv.Itoa(frame.Line) + "] - " + frame.Function[idxName+1:] + "()"
}

// GetCallerShort returns a Valuer that returns a file and line from a specified depth in the callstack.
func GetCallerShort(depth int) string {
    pc := make([]uintptr, 15)
    n := runtime.Callers(depth+1, pc)
    frame, _ := runtime.CallersFrames(pc[:n]).Next()
    //idxFile := strings.LastIndexByte(frame.File, '/')
    idx := strings.LastIndexByte(frame.Function, '/')
    idxName := strings.IndexByte(frame.Function[idx+1:], '.') + idx + 1

    return frame.Function[idxName+1:] + "()"
}

// GetTimestampStr format time
func GetTimestampStr() string {
    t := time.Now()
    return fmt.Sprintf("%d.%02d.%02d %02d:%02d:%02d-%d",
        t.Year(), t.Month(), t.Day(),
        t.Hour(), t.Minute(), t.Second(), t.Nanosecond())
}
