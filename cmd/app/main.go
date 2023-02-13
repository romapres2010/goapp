//go:build go1.19

package main

import (
	"context"
	"gopkg.in/yaml.v3"
	"os"
	"runtime/debug"

	"github.com/urfave/cli"

	_err "github.com/romapres2010/goapp/pkg/common/error"
	_log "github.com/romapres2010/goapp/pkg/common/logger"
	_metrics "github.com/romapres2010/goapp/pkg/common/metrics"
	_recover "github.com/romapres2010/goapp/pkg/common/recover"
	_version "github.com/romapres2010/goapp/pkg/common/version"

	_cfg "github.com/romapres2010/goapp/pkg/app/config"
	daemon "github.com/romapres2010/goapp/pkg/app/daemon"
)

// Параметры, подменяемые компилятором при сборке бинарника
var (
	version   = "unset"
	commit    = "unset"
	buildTime = "unset"
)

// Глобальные переменные, в которые парсятся входные флаги
var (
	argsConfigFile     string
	argsLogFile        string
	argsLogLevel       string
	argsHTTPListenSpec string
	argsHTTPUser       string
	argsHTTPPass       string
	argsJwtKey         string
	argsPgHost         string
	argsPgPort         string
	argsPgDBName       string
	argsPgUser         string
	argsPgPass         string
)

// Входные флаги программы
var flags = []cli.Flag{
	cli.StringFlag{
		Name:        "cfg",
		Usage:       "Config file name",
		Required:    false,
		Destination: &argsConfigFile,
	},
	cli.StringFlag{
		Name:        "listen, l",
		Usage:       "Listen string in format <host>:<port>",
		Required:    false,
		Destination: &argsHTTPListenSpec,
		//Value:       "localhost:3000",
	},
	cli.StringFlag{
		Name:        "httpuser",
		Usage:       "User name for access to HTTP server",
		Required:    false,
		Destination: &argsHTTPUser,
	},
	cli.StringFlag{
		Name:        "httppass",
		Usage:       "User password for access to HTTP server",
		Required:    false,
		Destination: &argsHTTPPass,
	},
	cli.StringFlag{
		Name:        "jwtkey",
		Usage:       "JSON web token secret key",
		Required:    false,
		Destination: &argsJwtKey,
	},
	cli.StringFlag{
		Name:        "loglevel",
		Usage:       "Log level: DEBUG, INFO, ERROR",
		Required:    false,
		Destination: &argsLogLevel,
	},
	cli.StringFlag{
		Name:        "logfile, log",
		Usage:       "Log file name",
		Required:    false,
		Destination: &argsLogFile,
	},
	cli.StringFlag{
		Name:        "pghost",
		Usage:       "DB host",
		Required:    false,
		Destination: &argsPgHost,
	},
	cli.StringFlag{
		Name:        "pgport",
		Usage:       "DB port",
		Required:    false,
		Destination: &argsPgPort,
	},
	cli.StringFlag{
		Name:        "pgdbname",
		Usage:       "DB name",
		Required:    false,
		Destination: &argsPgDBName,
	},
	cli.StringFlag{
		Name:        "pguser",
		Usage:       "DB user name",
		Required:    false,
		Destination: &argsPgUser,
	},
	cli.StringFlag{
		Name:        "pgpass",
		Usage:       "DB user password",
		Required:    false,
		Destination: &argsPgPass,
	},
}

// main function
func main() {
	var configFile string
	var exitCode int
	var app = cli.NewApp()

	_version.SetVersion(version, commit, buildTime)

	// Create new Application
	app.Name = "Example Go APP"
	app.Version = _version.Format()
	app.Author = "romapres"
	app.Email = "romapres@mail.ru"
	app.Flags = flags // присваиваем ранее определенные флаги
	app.Writer = os.Stderr

	// Определяем действие - запуск демона
	app.Action = func(ctx *cli.Context) (err error) {
		var appDaemon *daemon.Daemon

		// Функция восстановления после паники
		defer func() {
			r := recover()
			if r != nil {
				err = _recover.GetRecoverError(r, _err.ERR_UNDEFINED_ID)
			}
		}()

		// Создаем конфигурацию и подставляем данные из строки запуска и переменных окружения
		var globalCfg = &_cfg.Config{
			ArgsConfigFile:     argsConfigFile,
			ArgsHTTPListenSpec: argsHTTPListenSpec,
			ArgsHTTPUser:       argsHTTPUser,
			ArgsHTTPPass:       argsHTTPPass,
			ArgsJwtKey:         argsJwtKey,
			ArgsLogLevel:       argsLogLevel,
			ArgsLogFile:        argsLogFile,
			ArgsPgHost:         argsPgHost,
			ArgsPgPort:         argsPgPort,
			ArgsPgDBName:       argsPgDBName,
			ArgsPgUser:         argsPgUser,
			ArgsPgPass:         argsPgPass,

			EnvConfigFile:     os.Getenv(_cfg.ENV_APP_CONFIG_FILE),
			EnvHTTPListenSpec: os.Getenv(_cfg.ENV_APP_HTTP_LISTEN_SPEC),
			EnvHTTPUser:       os.Getenv(_cfg.ENV_APP_HTTP_USER),
			EnvHTTPPass:       os.Getenv(_cfg.ENV_APP_HTTP_PASS),
			EnvJwtKey:         os.Getenv(_cfg.ENV_APP_JWT_KEY),
			EnvLogLevel:       os.Getenv(_cfg.ENV_APP_LOG_LEVEL),
			EnvLogFile:        os.Getenv(_cfg.ENV_APP_LOG_FILE),
			EnvPgHost:         os.Getenv(_cfg.ENV_APP_PG_HOST),
			EnvPgPort:         os.Getenv(_cfg.ENV_APP_PG_PORT),
			EnvPgDBName:       os.Getenv(_cfg.ENV_APP_PG_DBNAME),
			EnvPgUser:         os.Getenv(_cfg.ENV_APP_PG_USER),
			EnvPgPass:         os.Getenv(_cfg.ENV_APP_PG_PASS),
		}

		{ // Считаем конфиг из файла
			if globalCfg.ArgsConfigFile != "" {
				_log.Info("Load config file 'configfile'='" + globalCfg.ArgsConfigFile + "' from command string")
				configFile = globalCfg.ArgsConfigFile
			} else if globalCfg.ArgsConfigFile == "" && globalCfg.EnvConfigFile != "" {
				_log.Info("Load config file 'configfile'='" + globalCfg.EnvConfigFile + "' from environment 'APP_CONFIG_FILE'")
				configFile = globalCfg.EnvConfigFile
			}

			if err = globalCfg.LoadYamlConfig(configFile); err != nil {
				return err
			}
		} // Считаем конфиг из файла

		// Настроим логирование
		if err = _log.SetGlobalConfig(globalCfg.LoggerCfg); err != nil {
			return err
		}

		// Конфиг выводим в log
		if globalCfgYaml, err := yaml.Marshal(&globalCfg); err != nil {
			return err
		} else {
			_log.Info("Configuration: \n", "\n"+string(globalCfgYaml))
		}

		// Настроим сбор метрик
		_metrics.InitGlobalMetrics(&globalCfg.MetricsCfg)

		// установить лимит на использование памяти
		if globalCfg.MemoryLimit != 0 {
			debug.SetMemoryLimit(globalCfg.MemoryLimit)
			_log.Info("Set Memory Limit: MemoryLimit", globalCfg.MemoryLimit)
		} else {
			_log.Info("Memory Limit was not set")
		}

		// Создаем демон
		if appDaemon, err = daemon.New(context.Background(), globalCfg); err != nil {
			return err
		}

		_log.Info("Server is startup: Version", app.Version)

		// Стартуем демон и ожидаем завершения
		return appDaemon.Run()
	}

	_log.Info("Starting up server")

	// Запускаем приложение
	if err := app.Run(os.Args); err != nil {
		_log.Error(err.Error()) // верхний уровень логирования с трассировкой
		exitCode = 1
	}

	_log.Info("Server was shutdown")

	// Останавливаем логирование
	_log.Info("Logger was shutting down")
	if err := _log.CloseLogger(); err != nil {
		_log.Error(err.Error()) // верхний уровень логирования с трассировкой
		exitCode = 1
	}

	os.Exit(exitCode)
}
