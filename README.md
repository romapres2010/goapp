# Шаблон backend сервера на Golang — часть 3 (Docker Compose Kubernetes Helm)

![Схема развертывания в Kubernetes](https://github.com/romapres2010/goapp/raw/master/doc/diagram/APP%20-%20Kebernates.jpg)

[Первая часть](https://habr.com/ru/post/492062/) шаблона была посвящена HTTP серверу.

[Вторая часть](https://habr.com/ru/post/500554/) шаблона была посвящена прототипированию REST API.

Третья часть посвящена развертыванию шаблона в Kubernetes и настройке Horizontal Autoscaler.

Для корректного развертывания в Kubernetes, в шаблон пришлось внести изменения: 
- способа конфигурирования - YAML, ENV, [Kustomize](https://kustomize.io/)
- подхода к логированию - переход на [zap](https://github.com/uber-go/zap)
- способа развертывания схемы БД - переход на [liquibase](https://www.liquibase.org/)

Ссылка на новый [репозиторий проекта](https://github.com/romapres2010/goapp).

Шаблон в репозитории представлен в виде полностью готового к развертыванию кода в Docker Composer или Kubernetes (helm).

## Содержание
1. Изменение подхода к конфигурированию
2. Изменение подхода к логированию
3. Развертывание схемы БД
4. Сборка Docker image
5. Сборка Docker-Compose 
6. Схема развертывания в Kubernetes
7. Подготовка YAML для Kubernetes
8. Kustomization YAML для Kubernetes
9. Подготовка Helm chart
11. Настройка Horizontal Autoscaler
 
<cut />

## 1. Изменение подхода к конфигурированию
Первоначальный вариант запуска приложения с передачей параметров через командную строку не очень удобен для Docker, так как усложняет работу с ENTRYPOINT.
Но для совместимости этот вариант приходится поддерживать.

Идеальным способом конфигурирования для Docker и Kubernetes являются переменные окружения ENV. Их проще всего подменять при переключении между средами DEV-TEST-PROD.  
В нашем случае количество конфигурационных параметров приложения перевалило за 1000 - поэтому конфиг был разделен на части: 
- условно постоянная часть, которая меняется редко - вынесена в YAML,
- все настройки, связанные со средами порты, пути, БД - вынесены в ENV,
- чувствительные настройки безопасности (пользователи, пароли) - вынесены в отдельное зашифрованное хранилище и передаются либо через ENV, либо через Secret.

### Условно постоянная часть конфига
Пример "базового" YAML конфига [app.global.yaml](https://github.com/romapres2010/goapp/blob/master/deploy/config/app.global.yaml).

При использовании YAML конфигов в Kubernetes, нужно решить пару задач:
- как сформировать YAML конфиг в зависимости от среды на которой будет развернуто приложение (DEV-TEST-PROD)
- как внедрить YAML конфиг в Docker контейнер, развернутый в Pod Kubernetes

Самый простой вариант с формированием YAML конфиг - это держать "базовую" версию и делать от нее "ручные клоны" для каждой из сред развертывания.
Самый простой вариант внедрения - это добавить YAML конфиги (всех сред) в сборку Docker образа и переключаться между ними через ENV переменные, в зависимости от среды на которой осуществляется запуск.

Более правильный вариант - при сборке использовать Kustomize и overlays для формирования из базового, YAML конфиг для конкретной среды и деплоить их в репозиторий артефактов, откуда он потом будет выложен в примонтированный каталог к Docker контейнер.

### Динамическое конфигурирование REST API entry point 
Столкнулись с тем, что в разных средах нужно по-разному настраивать REST API entry point. 
Можно было возложить эту функцию на nginx proxy, но решили задачу через YAML конфиг.
``` yaml
handlers:
    HealthHandler:  # Сервис health - проверка активности HEALTH
        enabled: true                      # Признак включен ли сервис
        application: "app"                 # Приложение к которому относится сервис
        module: "system"                   # Модуль к которому относится сервис
        service: "health"                  # Имя сервиса
        version: "v1.0.0"                  # Версия сервиса
        full_path: "/app/system/health"    # URI сервиса /Application.Module.Service.APIVersion или /Application/APIVersion/Module/Service
        params: ""                         # Параметры сервиса с виде {id:[0-9]+}
        method: "GET"                      # HTTP метод: GET, POST, ...
        handler_name: "HealthHandler"      # Имя функции обработчика
    ReadyHandler:   # Сервис ready - handle to test readinessProbe
        enabled: true                      # Признак включен ли сервис
        application: "app"                 # Приложение к которому относится сервис
        module: "system"                   # Модуль к которому относится сервис
        service: "ready"                   # Имя сервиса
        version: "v1.0.0"                  # Версия сервиса
        full_path: "/app/system/ready"     # URI сервиса /Application.Module.Service.APIVersion или /Application/APIVersion/Module/Service
        params: ""                         # Параметры сервиса с виде {id:[0-9]+}
        method: "GET"                      # HTTP метод: GET, POST, ...
        handler_name: "ReadyHandler"       # Имя функции обработчика
```

Назначение функции обработчика для REST API entry point выполнено [с использованием reflect](https://github.com/romapres2010/goapp/blob/master/pkg/common/httpservice/httpservice.go#L193).
Весьма удобный побочный эффект такого подхода - возможность переключать альтернативные обработчики без пересборки кода, только меняя конфиг.  

``` go
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
```

## 2. Изменение подхода к логированию
Логирование приложения в Kubernetes можно реализовать несколькими способами:
- вывод в stdout, stderr - этот способ является стандартным, но нужно учитывать, что в Kubernetes stdout, stderr Docker контейнеров перенаправляются в файлы, которые имеют ограниченный размер и могут перезаписываться. 
- вывод в файл на примонтированный к Docker контейнеру внешний ресурс.
- вывод в структурном виде во внешний сервис, например, Kafka.  

Всем этим требованиям отлично удовлетворяет библиотека [zap](https://github.com/uber-go/zap). Она также позволяет настроить несколько параллельных логгеров (ZapCore), которые будут писать в различные приемники.

В пакете [logger](https://github.com/romapres2010/goapp/tree/master/pkg/common/logger) реализована возможность настройки нескольких ZapCore через простой YAML конфиг.

Так же в пакете [logger](https://github.com/romapres2010/goapp/tree/master/pkg/common/logger) добавлена интеграция [zap](https://github.com/uber-go/zap) с библиотекой [lumberjack](https://gopkg.in/natefinch/lumberjack.v2) для управления циклом ротации и архивирования лог файлов. 

Любой из ZapCore можно динамически включать/выключать или изменять через HTTP POST: [/system/config/logger](https://github.com/romapres2010/goapp/blob/master/pkg/common/httpservice/handler_logger_set_config.go).

Например, в следующем конфиге объявлены 4 ZapCore.
- все сообщения от DEBUG до INFO выводятся в файл app.debug.log
  - максимальный размер лог файла в 10 MB
  - время хранения истории лог файлов 7 дней
  - максимальное количество архивных логов - 10 без архивирования
- все сообщения от ERROR до FATAL выводятся в файл app.error.log
- все сообщения выводятся в stdout
- все сообщения от ERROR до FATAL выводятся в stderr

``` yaml
# конфигурация сервиса логирования
logger:
    enable: true                        # состояние логирования 'true', 'false'
    global_level: INFO                  # debug, info, warn, error, dpanic, panic, fatal - все логгеры ниже этого уровня будут отключены
    global_filename: /app/log/app.log   # глобальное имя файл для логирования
    zap:
        enable: true                # состояние логирования 'true', 'false'
        disable_caller: false       # запретить вывод в лог информации о caller
        disable_stacktrace: false   # запретить вывод stacktrace
        development: false          # режим разработки для уровня dpanic
        stacktrace_level: error     # для какого уровня выводить stacktrace debug, info, warn, error, dpanic, panic, fatal
        core:
            -   enable: true        # состояние логирования 'true', 'false'
                min_level: null     # минимальный уровень логирования debug, info, warn, error, dpanic, panic, fatal
                max_level: INFO     # максимальный уровень логирования debug, info, warn, error, dpanic, panic, fatal
                log_to: lumberjack  # логировать в 'file', 'stderr', 'stdout', 'url', 'lumberjack'
                encoding: "console" # формат вывода 'console', 'json'
                file:
                    filename: ".debug.log"       # имя файл для логирования, если не заполнено, то используется глобальное имя
                    max_size: 10                 # максимальный размер лог файла в MB
                    max_age: 7                   # время хранения истории лог файлов в днях
                    max_backups: 10              # максимальное количество архивных логов
                    local_time: true             # использовать локальное время в имени архивных лог файлов
                    compress: false              # сжимать архивные лог файлы в zip архив
            -   enable: true        # состояние логирования 'true', 'false'
                min_level: ERROR    # минимальный уровень логирования debug, info, warn, error, dpanic, panic, fatal
                max_level: null     # максимальный уровень логирования debug, info, warn, error, dpanic, panic, fatal
                log_to: lumberjack  # логировать в 'file', 'stderr', 'stdout', 'url', 'lumberjack
                encoding: "console" # формат вывода 'console', 'json'
                file:
                    filename: ".error.log"       # имя файл для логирования, если не заполнено, то используется глобальное имя
                    max_size: 10                 # максимальный размер лог файла в MB
                    max_age: 7                   # время хранения истории лог файлов в днях
                    max_backups: 10              # максимальное количество архивных логов
                    local_time: true             # использовать локальное время в имени архивных лог файлов
                    compress: false              # сжимать архивные лог файлы в zip архив
            -   enable: true        # состояние логирования 'true', 'false'
                min_level: null     # минимальный уровень логирования debug, info, warn, error, dpanic, panic, fatal
                max_level: null     # максимальный уровень логирования debug, info, warn, error, dpanic, panic, fatal
                log_to: stdout      # логировать в 'file', 'stderr', 'stdout', 'url', 'lumberjack
                encoding: "console" # формат вывода 'console', 'json'
            -   enable: true        # состояние логирования 'true', 'false'
                min_level: ERROR    # минимальный уровень логирования debug, info, warn, error, dpanic, panic, fatal
                max_level: null     # максимальный уровень логирования debug, info, warn, error, dpanic, panic, fatal
                log_to: stderr      # логировать в 'file', 'stderr', 'stdout', 'url', 'lumberjack
                encoding: "console" # формат вывода 'console', 'json'
```

## 3. Развертывание схемы БД
Для первоначального развертывания схемы БД в контейнере и управления изменениями DDL скриптов через Git великолепно подходит [liquibase](https://www.liquibase.org/).

[Liquibase](https://www.liquibase.org/) позволяет самостоятельно генерировать DDL скрипты БД. Для простых случаев - этого вполне хватает, но если требуется тонкая настройка БД (например,настройка партиций, Blob storage) то лучше формировать DDL скрипты в отдельном приложении. 
Для Postgres отлично подходит TOAD DataModeler. Для Oracle - Oracle SQL Developer Data Modeler. Например, TOAD DataModeler умеет корректно генерировать DDL скрипты для добавления not null столбцов в уже заполненную таблицу. 

Весьма полезная функция [Liquibase](https://www.liquibase.org/) - возможность задавать скрипты для отката изменений. Если установка патча на БД прошла неуспешно (сбой добавления столбца в таблицу, нарушение PK, FK), то легко откатиться к предыдущему состоянию.  

В шаблон включены примеры [liquibase/changelog](https://github.com/romapres2010/goapp/tree/master/db/liquibase/changelog) и [DDL скрипты](https://github.com/romapres2010/goapp/tree/master/db/sql) для таблиц Country и Currency с минимальным тестовым наполнением. 

Так же как и с YAML конфигами, необходимо предусмотреть способ внедрения [Liquibase](https://www.liquibase.org/) Changelog в Docker контейнер Liquibase, развернутый в Job Pod Kubernetes:
- скрипты можно встраивать в Liquibase контейнер (не очень хороший вариант, как минимум размер Docker с Liquibase более 300 Мбайт), 
- скрипты можно выкладывать в примонтированный каталог к Docker контейнер и на уровне ENV переменных указывать Changelog для запуска.

В шаблоне предусмотрены оба эти варианта.

## 4. Сборка Docker image
Docker файлы для сборки выложены в каталоге [deploy/docker](https://github.com/romapres2010/goapp/tree/master/deploy/docker).

Для тестирования сборки можно использовать Docker Desktop - в этом случае не нужно делать push Docker image - они все кешируются на локальной машине.

### Сборка Docker image для Go Api

Сборка Docker image для Go Api [app-api.Dockerfile](https://github.com/romapres2010/goapp/blob/master/deploy/docker/app-api.Dockerfile) имеет следующие особенности:
- git commit, время сборки и базовая версия приложения передаются через аргументы docker
- создаются точки монтирования для внешних и преднастроенных YAML конфигов и log файлов
- сборка ведется только на локальной копии внешних библиотек ./vendor
- преднастроенные YAML конфиги для различных сред DEV-TEST-PROD можно встроить в сборку и переключаться через ENV переменные
- git commit, время сборки и базовая версия приложения встраиваются в пакет main
- для финальной сборки используется минимальный distroless образ
- точка запуска приложения не содержит параметров - все передается через ENV переменные

``` docker
##
## Build stage
##

FROM golang:1.19-buster AS build

# git commit, время сборки и базовая версия приложения передаются через аргументы docker
ARG APP_COMMIT
ARG APP_BUILD_TIME
ARG APP_VERSION

WORKDIR /app

# создаются точки монтирования для внешних и преднастроенных YAML конфигов и log файлов
RUN mkdir ./run && mkdir ./run/defcfg && mkdir ./run/log && mkdir ./run/cfg

COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
COPY ./pkg ./pkg
COPY ./cmd/app/main.go ./cmd/app/main.go

# сборка ведется только на локальной копии внешних библиотек ./vendor
COPY ./vendor ./vendor

# преднастроенные YAML конфиги для различных сред DEV-TEST-PROD можно встроить в сборку и переключаться через ENV переменные
COPY ./deploy/config/. ./run/defcfg/

# git commit, время сборки и базовая версия приложения встраиваются в пакет main
RUN go build -v -mod vendor -ldflags "-X main.commit=${APP_COMMIT} -X main.buildTime=${APP_BUILD_TIME} -X main.version=${APP_VERSION}" -o ./run/main ./cmd/app/main.go

RUN echo "Based on commit: $APP_COMMIT" && echo "Build Time: $APP_BUILD_TIME" && echo "Version: $APP_VERSION"

##
## Deploy stage
##
FROM gcr.io/distroless/base-debian10

WORKDIR /app

COPY --from=build /app/run/. .

EXPOSE 8080/tcp

# точка запуска приложения не содержит параметров - все передается через ENV переменные
ENTRYPOINT [ "/app/main"]
```

Для ручной сборки Docker выложен тестовый shell скрипт [app-api-build.sh](https://github.com/romapres2010/goapp/blob/master/deploy/docker/app-api-build.sh):
- сборка и запуск всех скриптов всегда осуществляется от корня репозитория
- логирование вывода всех скриптов идет в файлы
- управление версией, именем приложения и docker tag представлено в упрощенном виде через локальные файлы в каталоге [deploy](https://github.com/romapres2010/goapp/tree/master/deploy)
  - базовая версия приложения ведется в текстовом файле [version](https://github.com/romapres2010/goapp/blob/master/deploy/version). Для CI/CD через github action дополнительно используется сквозная нумерация, которая добавляется к базовой версии. 
  - имя приложения ведется в текстовом файле [app_api_app_name](https://github.com/romapres2010/goapp/blob/master/deploy/app_api_app_name)ведется в текстовом файле [app_api_app_name](https://github.com/romapres2010/goapp/blob/master/deploy/app_api_app_name)
  - имя репозитория для публикации ведется в текстовом файле [default_repository](https://github.com/romapres2010/goapp/blob/master/deploy/default_repository)

``` shell
echo "Current directory:"
pwd

export LOG_FILE=$(pwd)/app-api-build.log

echo "Go to working directory:"
pushd ./../../

export APP_VERSION=$(cat ./deploy/version)
export APP_API_APP_NAME=$(cat ./deploy/app_api_app_name)
export APP_REPOSITORY=$(cat ./deploy/default_repository)
export APP_BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
export APP_COMMIT=$(git rev-parse --short HEAD)

docker build --build-arg APP_VERSION=$APP_VERSION --build-arg APP_BUILD_TIME=$APP_BUILD_TIME --build-arg APP_COMMIT=$APP_COMMIT -t $APP_REPOSITORY/$APP_API_APP_NAME:$APP_VERSION -f ./deploy/docker/app-api.Dockerfile . 1>$LOG_FILE 2>&1

echo "Go to current directory:"
popd
```

### Сборка Docker image для liquibase
Сборка Docker image для Liquibase [liquibase.Dockerfile](https://github.com/romapres2010/goapp/blob/master/deploy/docker/app-liquibase.Dockerfile) имеет следующие особенности:
- Changelog для различных сред DEV-TEST-PROD можно встроить в сборку и переключаться через ENV переменные
- DDL и DML скрипты можно встроить в сборку либо использовать внешнюю точку монтирования

``` dockerfile
##
## Deploy stage
##
FROM liquibase/liquibase:4.9.1

# Changelog для различных сред DEV-TEST-PROD можно встроить в сборку и переключаться через ENV переменные
COPY ./db/liquibase/changelog/. /liquibase/changelog/

# DDL и DML скрипты можно встроить в сборку
COPY ./db/sql /sql

```

## 5. Сборка Docker Compose
Для локальной сборки удобно использовать Docker Compose. Скрипт сборки [compose-app.yaml](https://github.com/romapres2010/goapp/blob/master/deploy/compose/compose-app.yaml).

В каталоге [/deploy/run/local.compose.global.config](https://github.com/romapres2010/goapp/tree/master/deploy/run/local.compose.global.config) выложены скрипты для локального тестирования с использованием Docker Compose.
- все ENV переменные приложений передаются через внешний [.env файл](https://github.com/romapres2010/goapp/blob/master/deploy/run/local.compose.global.config/.env)
- каждый компонент системы (API, БД, UI) можно поднимать независимо, например:
  - [/deploy/run/local.compose.global.config/compose-app-api-db-reload-up.sh](https://github.com/romapres2010/goapp/blob/master/deploy/run/local.compose.global.config/compose-app-api-db-reload-up.sh) - полностью чистит БД, и пересобирает API
  - [/deploy/run/local.compose.global.config/compose-app-ui-up.sh](https://github.com/romapres2010/goapp/blob/master/deploy/run/local.compose.global.config/compose-app-ui-up.sh) - переустанавливает только UI

### Docker Compose для Go Api
Особенности:
- при каждом запуске выполняется build
- аргументы в Dockerfile (APP_COMMIT, APP_BUILD_TIME, APP_VERSION) пробрасываются через ENV переменные окружения
- ENV переменные приложения берутся из одноименных ENV переменных окружения (дефолтные значения указаны для примера)
- в Docker монтируются volumеs для конфиг и логов (/app/cfg:ro, /app/log:rw)
- зависимости (depends_on) от БД не выставляются - вместо этого приложение написано так, чтобы оно могло перестартовывать без последствий и указано restart_policy: on-failure
- настроен healthcheck на GET: /app/system/health

``` dockerfile
  app-api:
    build:
      context: ./../../
      dockerfile: ./deploy/docker/app-api.Dockerfile
      tags:
        - $APP_REPOSITORY/$APP_API_APP_NAME:$APP_VERSION
      args:
        - APP_COMMIT=${APP_COMMIT:-unset}
        - APP_BUILD_TIME=${APP_BUILD_TIME:-unset}
        - APP_VERSION=${APP_VERSION:-unset}
    container_name: app-api
    hostname: app-api-host
    networks:
      - app-net
    ports:
      - $APP_HTTP_OUT_PORT:$APP_HTTP_PORT
      - 8001:$APP_HTTP_PORT
    environment:
      - TZ="Europe/Moscow"
      - APP_CONFIG_FILE=${APP_CONFIG_FILE:-/app/defcfg/app.global.yaml}
      - APP_HTTP_LISTEN_SPEC=${APP_HTTP_LISTEN_SPEC:-0.0.0.0:8080}
      - APP_LOG_LEVEL=${APP_LOG_LEVEL:-ERROR}
      - APP_LOG_FILE=${APP_LOG_FILE:-/app/log/app.log}
      - APP_PG_USER=${APP_PG_USER:-postgres}
      - APP_PG_PASS=${APP_PG_PASS:?database password not set}
      - APP_PG_HOST=${APP_PG_HOST:-app-db-host}
      - APP_PG_PORT=${APP_PG_PORT:-5432}
      - APP_PG_DBNAME=${APP_PG_DBNAME:-postgres}
    volumes:
      - "./../../../app_volumes/cfg:/app/cfg:ro"
      - "./../../../app_volumes/log:/app/log:rw"
    deploy:
      restart_policy:
        condition: on-failure
    healthcheck:
      test: ["curl -f 0.0.0.0:8080/app/system/health"]
      interval: 10s
      timeout: 5s
      retries: 5
```

### Docker Compose для liquibase
Особенности:
- при необходимости тестирования сборки подмнтируется дополнительный Volume для логов (/liquibase/mylog:rw)  
- устанавливается зависимость от БД (depends_on condition: service_healthy)
- при необходимости в Docker монтируются volumеs для changelog и sql (/liquibase/sql:rw, /liquibase/changelog:rw)
- changelog для запуска передается через ENV переменные

``` dockerfile
  app-liquibase:
    build:
      context: ./../../
      dockerfile: ./deploy/docker/app-liquibase.Dockerfile
      tags:
        - $APP_REPOSITORY/$APP_LUQUIBASE_APP_NAME:$APP_VERSION
    container_name: app-liquibase
    depends_on:
      app-db:
        condition: service_healthy
    networks:
      - app-net
#    volumes:
#      - "./../../../app_volumes/log:/liquibase/mylog:rw"
#      - "./../../../app_volumes/sql:/liquibase/sql:rw"
#      - "./../../../app_volumes/log:/liquibase/changelog:rw"
#    command: --changelog-file=./changelog/$APP_PG_CHANGELOG --url="jdbc:postgresql://$APP_PG_HOST:$APP_PG_PORT/$APP_PG_DBNAME" --username=$APP_PG_USER --password=$APP_PG_PASS --logFile="mylog/liquibase.log" --logLevel=info update
    command: --changelog-file=./changelog/$APP_PG_CHANGELOG --url="jdbc:postgresql://$APP_PG_HOST:$APP_PG_PORT/$APP_PG_DBNAME" --username=$APP_PG_USER --password=$APP_PG_PASS --logLevel=info update
```

### Docker Compose для БД
Особенности:
- в Docker монтируются volumе для БД (/var/lib/postgresql/data:rw)
- настроен healthcheck на "CMD-SHELL", "pg_isready"

``` dockerfile
  app-db:
    image: postgres:14.5-alpine
    container_name: app-db
    hostname: app-db-host
    environment:
      - POSTGRES_PASSWORD=${APP_PG_PASS:?database password not set}
      - PGUSER=${APP_PG_USER:?database user not set}
    networks:
      - app-net
    ports:
      - $APP_PG_OUT_PORT:$APP_PG_PORT
    volumes:
      - "./../../../app_volumes/db:/var/lib/postgresql/data:rw"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 10s
    restart: unless-stopped
```


## 6. Схема развертывания в Kubernetes
Схема развертывания приведена в сокращенном виде, исключены компоненты, связанные с UI (Jmix) и распределенным кэшем (hazelcast).
- Все компоненты развертываются в отдельном namespace (subnamespace)
- API
  - 
- БД
  - 

![Схема развертывания в Kubernetes](https://github.com/romapres2010/goapp/raw/master/doc/diagram/APP%20-%20Kebernates.jpg)

## 7. Подготовка YAML для Kubernetes


## . Подготовка Helm chart

## . Настройка Horizontal Autoscaler
Управление памятью в контейнере - memory_limit: 2000000000 - как срабатывает снижение количества replica


Так же нужно иметь в виду, что в Kubernetes может быть запущено более одного экземпляра Docker контейнера и так же возможна ситуация, когда Docker контейнер будет удален и пересоздано на другом узле кластера.
