package config

import (
    "gopkg.in/yaml.v3"
    "os"

    _err "github.com/romapres2010/goapp/pkg/common/error"
    _httplog "github.com/romapres2010/goapp/pkg/common/httplog"
    _httpserver "github.com/romapres2010/goapp/pkg/common/httpserver"
    _http "github.com/romapres2010/goapp/pkg/common/httpservice"
    _log "github.com/romapres2010/goapp/pkg/common/logger"
    _metrics "github.com/romapres2010/goapp/pkg/common/metrics"

    httphandler "github.com/romapres2010/goapp/pkg/app/httphandler"
)

const (
    ENV_APP_CONFIG_FILE      = "APP_CONFIG_FILE"
    ENV_APP_HTTP_LISTEN_SPEC = "APP_HTTP_LISTEN_SPEC"
    ENV_APP_HTTP_USER        = "APP_HTTP_USER"
    ENV_APP_HTTP_PASS        = "APP_HTTP_PASS"
    ENV_APP_LOG_LEVEL        = "APP_LOG_LEVEL"
    ENV_APP_LOG_FILE         = "APP_LOG_FILE"
    ENV_APP_JWT_KEY          = "APP_JWT_KEY"
    ENV_APP_PG_HOST          = "APP_PG_HOST"
    ENV_APP_PG_PORT          = "APP_PG_PORT"
    ENV_APP_PG_DBNAME        = "APP_PG_DBNAME"
    ENV_APP_PG_USER          = "APP_PG_USER"
    ENV_APP_PG_PASS          = "APP_PG_PASS"
)

// Config represent daemon options
type Config struct {
    MemoryLimit int64  `yaml:"memory_limit" json:"memory_limit"` // рекомендуемый предел памяти в байтах - использовать для запуска в контейнере
    ConfigFile  string `yaml:"config_file" json:"config_file"`   // основной файл конфигурации

    ArgsConfigFile     string `yaml:"args_config_file" json:"args_config_file"`           // основной файл конфигурации
    ArgsHTTPListenSpec string `yaml:"args_http_listen_spec" json:"args_http_listen_spec"` // строка HTTP листенера из командной строки
    ArgsHTTPUser       string `yaml:"args_http_user" json:"args_http_user"`               // пользователь для HTTP Basic Authentication из командной строки
    ArgsHTTPPass       string `yaml:"args_http_pass" json:"args_http_pass"`               // пароль для HTTP Basic Authentication из командной строки
    ArgsJwtKey         string `yaml:"args_jwt_key" json:"args_jwt_key"`                   // JWT secret key из командной строки
    ArgsLogLevel       string `yaml:"args_log_level" json:"args_log_level"`               // уровень логирования из командной строки
    ArgsLogFile        string `yaml:"args_log_file" json:"args_log_file"`                 // расположение log файла
    ArgsPgHost         string `yaml:"args_pg_host" json:"args_pg_host"`                   // host БД
    ArgsPgPort         string `yaml:"args_pg_port" json:"args_pg_port"`                   // порт БД
    ArgsPgDBName       string `yaml:"args_pg_dbname" json:"args_pg_dbname"`               // имя БД
    ArgsPgUser         string `yaml:"args_pg_user" json:"args_pg_user"`                   // пользователь к БД
    ArgsPgPass         string `yaml:"args_pg_pass" json:"args_pg_pass"`                   // пароль пользователя БД

    EnvConfigFile     string `yaml:"env_config_file" json:"env_config_file"`           // основной файл конфигурации
    EnvHTTPListenSpec string `yaml:"env_http_listen_spec" json:"env_http_listen_spec"` // строка HTTP листенера из командной строки
    EnvHTTPUser       string `yaml:"env_http_user" json:"env_http_user"`               // пользователь для HTTP Basic Authentication из командной строки
    EnvHTTPPass       string `yaml:"env_http_pass" json:"env_http_pass"`               // пароль для HTTP Basic Authentication из командной строки
    EnvJwtKey         string `yaml:"env_jwt_key" json:"env_jwt_key"`                   // JWT secret key из командной строки
    EnvLogLevel       string `yaml:"env_log_level" json:"env_log_level"`               // уровень логирования из командной строки
    EnvLogFile        string `yaml:"env_log_file" json:"env_log_file"`                 // расположение log файла
    EnvPgHost         string `yaml:"env_pg_host" json:"env_pg_host"`                   // host БД
    EnvPgPort         string `yaml:"env_pg_port" json:"env_pg_port"`                   // порт БД
    EnvPgDBName       string `yaml:"env_pg_dbname" json:"env_pg_dbname"`               // имя БД
    EnvPgUser         string `yaml:"env_pg_user" json:"env_pg_user"`                   // пользователь к БД
    EnvPgPass         string `yaml:"env_pg_pass" json:"env_pg_pass"`                   // пароль пользователя БД

    // Конфигурация вложенных сервисов
    LoggerCfg      _log.Config        `yaml:"logger" json:"logger"`             // конфигурация сервиса логирования
    HttpServerCfg  _httpserver.Config `yaml:"http_server" json:"http_server"`   // конфигурация HTTP сервера
    HttpServiceCfg _http.Config       `yaml:"http_service" json:"http_service"` // конфигурация обработчиков HTTP запросов
    HttpHandlerCfg httphandler.Config `yaml:"http_handler" json:"http_handler"` // конфигурация обработчиков HTTP запросов
    HttpLoggerCfg  _httplog.Config    `yaml:"http_logger" json:"http_logger"`   // конфигурация сервиса логирования HTTP трафика
    MetricsCfg     _metrics.Config    `yaml:"metrics" json:"metrics"`           // конфигурация сбора метрик
}

// ValidateConfig validate config
func (cfg *Config) ValidateConfig() error {

    if !(cfg.LoggerCfg.GlobalLevel == _log.LEVEL_DEBUG || cfg.LoggerCfg.GlobalLevel == _log.LEVEL_ERROR || cfg.LoggerCfg.GlobalLevel == _log.LEVEL_INFO) {
        return _err.NewTyped(_err.ERR_INCORRECT_LOG_LEVEL, _err.ERR_UNDEFINED_ID, cfg.LoggerCfg.GlobalLevel).PrintfError()
    }

    // задан ли в командной строке JSON web token secret key
    if cfg.HttpServiceCfg.UseJWT && cfg.ArgsJwtKey == "" {
        return _err.NewTyped(_err.ERR_EMPTY_JWT_KEY, _err.ERR_UNDEFINED_ID).PrintfError()
    }

    // задан ли в командной строке JSON web token secret key
    if cfg.HttpServiceCfg.AuthType == _http.AUTH_TYPE_INTERNAL && (cfg.ArgsHTTPUser == "" || cfg.ArgsHTTPPass == "") {
        return _err.NewTyped(_err.ERR_EMPTY_HTTP_USER, _err.ERR_UNDEFINED_ID).PrintfError()
    }

    if !(cfg.HttpServiceCfg.AuthType == _http.AUTH_TYPE_NONE || cfg.HttpServiceCfg.AuthType == _http.AUTH_TYPE_MSAD || cfg.HttpServiceCfg.AuthType == _http.AUTH_TYPE_INTERNAL) {
        return _err.NewTyped(_err.ERR_INCORRECT_AUTH_TYPE, _err.ERR_UNDEFINED_ID, cfg.HttpServiceCfg.AuthType).PrintfError()
    }

    return nil
}

func replaceCfgArgsEnv(cfgParamName string, cfgParam *string, argsParamName string, argsParam string, envParamName string, envParam string) {
    if argsParam != "" {
        _log.Info("YAML config parameter '" + cfgParamName + "'='" + *cfgParam + "' was replaced with parameter '" + argsParamName + "'='" + argsParam + "' from command string")
        *cfgParam = argsParam
    } else if argsParam == "" && envParam != "" {
        _log.Info("YAML config parameter '" + cfgParamName + "'='" + *cfgParam + "' was replaced with environment '" + envParamName + "'='" + envParam + "'")
        *cfgParam = envParam
    }
}

// LoadYamlConfig load configuration file
func (cfg *Config) LoadYamlConfig(fileName string) error {
    if cfg != nil {
        var err error
        var fileInfo os.FileInfo
        var file *os.File

        cfg.ConfigFile = fileName

        if fileName == "" {
            return _err.NewTyped(_err.ERR_EMPTY_CONFIG_FILE, _err.ERR_UNDEFINED_ID).PrintfError()
        }
        _log.Info("Loading config from file: FileName", fileName)

        // Считать информацию о файле
        fileInfo, err = os.Stat(fileName)
        if os.IsNotExist(err) {
            return _err.NewTyped(_err.ERR_CONFIG_FILE_NOT_EXISTS, _err.ERR_UNDEFINED_ID, fileName).PrintfError()
        }

        _log.Info("Config file exist: FileName, FileInfo", fileName, fileInfo)

        { // Считать конфигурацию из файла
            file, err = os.Open(fileName)
            if err != nil {
                return err
            }
            defer func() {
                if file != nil {
                    err = file.Close()
                    if err != nil {
                        _ = _err.WithCauseTyped(_err.ERR_COMMON_ERROR, _err.ERR_UNDEFINED_ID, err, "config.LoadYamlConfig -> os.File.Close()").PrintfError()
                    }
                }
            }()

            decoder := yaml.NewDecoder(file)
            err = decoder.Decode(cfg)

            if err != nil {
                return _err.WithCauseTyped(_err.ERR_CONFIG_FILE_LOAD_ERROR, _err.ERR_UNDEFINED_ID, err, fileName).PrintfError()
            }
        } // Считать конфигурацию из файла

        { // подменить параметры из окружения и командной строки, приоритет за командной строкой
            replaceCfgArgsEnv(
                "http_server.listen_spec", &cfg.HttpServerCfg.ListenSpec,
                "listenspec", cfg.ArgsHTTPListenSpec,
                ENV_APP_HTTP_LISTEN_SPEC, cfg.EnvHTTPListenSpec)

            replaceCfgArgsEnv(
                "http_service.http_user", &cfg.HttpServiceCfg.HTTPUser,
                "httpuser", cfg.ArgsHTTPUser,
                ENV_APP_HTTP_USER, cfg.EnvHTTPUser)

            replaceCfgArgsEnv(
                "http_service.http_pass", &cfg.HttpServiceCfg.HTTPPass,
                "httppass", cfg.ArgsHTTPPass,
                ENV_APP_HTTP_PASS, cfg.EnvHTTPPass)

            replaceCfgArgsEnv(
                "http_service.jwt_key", &cfg.HttpServiceCfg.JwtKeyStr,
                "jwt_key", cfg.ArgsJwtKey,
                ENV_APP_JWT_KEY, cfg.EnvJwtKey)
            cfg.HttpServiceCfg.JwtKey = []byte(cfg.HttpServiceCfg.JwtKeyStr)

            replaceCfgArgsEnv(
                "logger.global_level", &cfg.LoggerCfg.GlobalLevel,
                "loglevel", cfg.ArgsLogLevel,
                ENV_APP_LOG_LEVEL, cfg.EnvLogLevel)

            replaceCfgArgsEnv(
                "logger.log_file", &cfg.LoggerCfg.GlobalFilename,
                "logfile", cfg.ArgsLogFile,
                ENV_APP_LOG_FILE, cfg.EnvLogFile)

        } // подменить параметры из окружения и командной строки, приоритет за командной строкой

        return cfg.ValidateConfig()
    }
    return _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, _err.ERR_UNDEFINED_ID, "if cfg != nil {}").PrintfError()
}
