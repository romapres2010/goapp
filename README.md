# [Шаблон backend сервера на Golang — часть 3 (Docker, Docker Compose, Kubernetes (kustomize)](https://habr.com/ru/post/716634/)

![Схема развертывания в Kubernetes](https://github.com/romapres2010/goapp/raw/master/doc/diagram/APP%20-%20Kebernates.jpg)

[Первая часть](https://habr.com/ru/post/492062/) шаблона была посвящена HTTP серверу.

[Вторая часть](https://habr.com/ru/post/500554/) шаблона была посвящена прототипированию REST API.

[Третья часть](https://habr.com/ru/post/716634/) посвящена развертыванию шаблона в Docker, Docker Compose, Kubernetes (kustomize).

Четвертая часть будет посвящена развертыванию в Kubernetes с Helm chart и настройке Horizontal Autoscaler.

[Пятая часть](https://habr.com/ru/post/720286/) посвящена оптимизации Worker pool и особенностям его работы в составе микросервиса, развернутого в Kubernetes.

Для корректного развертывания в Kubernetes, в шаблон пришлось внести изменения: 
- способа конфигурирования - YAML, ENV, [Kustomize](https://kustomize.io/)
- подхода к логированию - переход на [zap](https://github.com/uber-go/zap)
- способа развертывания схемы БД - переход на [liquibase](https://www.liquibase.org/)
- добавление метрик [prometheus](https://prometheus.io/)

Ссылка на новый [репозиторий](https://github.com/romapres2010/goapp).

Шаблон goapp в репозитории полностью готов к развертыванию в Docker, Docker Compose, Kubernetes (kustomize), Kubernetes (helm).

_Настоящая статья не содержит детального описание используемых технологий_ 

## Содержание
1. Изменение подхода к конфигурированию
2. Добавление метрик [prometheus](https://prometheus.io/)
3. Изменение подхода к логированию
4. Развертывание схемы БД
5. Сборка Docker image
6. Сборка Docker-Compose 
7. Схема развертывания в Kubernetes
8. Подготовка YAML для Kubernetes
9. Kustomization YAML для Kubernetes
10. Тестирование Kubernetes с kustomize
 
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

## 2 Добавление метрик [prometheus](https://prometheus.io/)

В шаблон встроена сборка следующих метрик для [prometheus](https://prometheus.io/):
- Метрики DB
  - **db_total** - The total number of processed DB by sql statement (CounterVec)
  - **db_duration_ms** - The duration histogram of DB operation in ms by sql statement (HistogramVec)
- Метрики HTTP request
  - **http_requests_total_by_resource** - How many HTTP requests processed, partitioned by resource (CounterVec)
  - **http_requests_error_total_by_resource** - How many HTTP requests was ERRORED, partitioned by resource (CounterVec)
  - **http_request_duration_ms_by_resource** - The duration histogram of HTTP requests in ms by resource (HistogramVec)
  - **http_active_requests_count** - The total number of active HTTP requests (Gauge)
  - **http_request_duration_ms** - The duration histogram of HTTP requests in ms (Histogram)
- Метрики HTTP client call
  - **http_client_call_total_by_resource** - How many HTTP client call processed, partitioned by resource (CounterVec)
  - **http_client_call_duration_ms_by_resource** - The duration histogram of HTTP client call in ms by resource (HistogramVec)
- Метрики WorkerPool
  - **wp_task_queue_buffer_len_vec** - The len of the worker pool buffer (GaugeVec)
  - **wp_add_task_wait_count_vec** - The number of the task waiting to add to worker pool queue (GaugeVec)
  - **wp_worker_process_count_vec** - The number of the working worker (GaugeVec)

Доступ к метрикам настроен по стандартному пути HTTP GET [/metrics](http://127.0.0.1:3000/metrics).

Включение / выключение метрик настраивается через YAML конфиг 
``` yaml
# конфигурация сбора метрик
metrics:
    metrics_namespace: com
    metrics_subsystem: go_app
    collect_db_count_vec: true
    collect_db_duration_vec: false
    collect_http_requests_count_vec: true
    collect_http_error_requests_count_vec: true
    collect_http_requests_duration_vec: true
    collect_http_active_requests_count: true
    collect_http_requests_duration: true
    collect_http_client_call_count_vec: true
    collect_http_client_call_duration_vec: true
    collect_wp_task_queue_buffer_len_vec: true
    collect_wp_add_task_wait_count_vec: true
    collect_wp_worker_process_count_vec: true
```

## 3. Изменение подхода к логированию
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

## 4. Развертывание схемы БД
Для первоначального развертывания схемы БД в контейнере и управления изменениями DDL скриптов через Git великолепно подходит [liquibase](https://www.liquibase.org/).

[Liquibase](https://www.liquibase.org/) позволяет самостоятельно генерировать DDL скрипты БД. Для простых случаев - этого вполне хватает, но если требуется тонкая настройка БД (например,настройка партиций, Blob storage) то лучше формировать DDL скрипты в отдельном приложении. 
Для Postgres отлично подходит TOAD DataModeler. Для Oracle - Oracle SQL Developer Data Modeler. Например, TOAD DataModeler умеет корректно генерировать DDL скрипты для добавления not null столбцов в уже заполненную таблицу. 

Весьма полезная функция [Liquibase](https://www.liquibase.org/) - возможность задавать скрипты для отката изменений. Если установка патча на БД прошла неуспешно (сбой добавления столбца в таблицу, нарушение PK, FK), то легко откатиться к предыдущему состоянию.  

В шаблон включены примеры [liquibase/changelog](https://github.com/romapres2010/goapp/tree/master/db/liquibase/changelog) и [DDL скрипты](https://github.com/romapres2010/goapp/tree/master/db/sql) для таблиц Country и Currency с минимальным тестовым наполнением. 

Так же как и с YAML конфигами, необходимо предусмотреть способ внедрения [Liquibase](https://www.liquibase.org/) Changelog в Docker контейнер Liquibase, развернутый в Job Pod Kubernetes:
- скрипты можно встраивать в Liquibase контейнер (не очень хороший вариант, как минимум размер Docker с Liquibase более 300 Мбайт), 
- скрипты можно выкладывать в примонтированный каталог к Docker контейнер и на уровне ENV переменных указывать Changelog для запуска.

В шаблоне предусмотрены оба эти варианта.

## 5. Сборка Docker image
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

## 6. Сборка Docker Compose
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
- при необходимости тестирования сборки можно примонтировать дополнительный Volume для логов (/liquibase/mylog:rw)  
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

## 7. Схема развертывания в Kubernetes
Схема развертывания в [Kubernetes](https://kubernetes.io/ru/docs/tutorials/kubernetes-basics/) приведена в сокращенном виде, исключены компоненты, связанные с UI (Jmix) и распределенным кэшем (hazelcast).

![Схема развертывания в Kubernetes](https://github.com/romapres2010/goapp/raw/master/doc/diagram/APP%20-%20Kebernates.jpg)

Основным элементом развертывания в Kubernetes является [Pod](https://kubernetes.io/docs/concepts/workloads/pods/). 
Для простоты - это группа контейнеров с общими ресурсами, которая изолирована от остальных Pod.
- Обычно, в Pod размещается один app контейнер, и, при необходимости, init контейнер.
- Остановить app контейнер в Pod нельзя, либо он завершит работу сам, либо нужно удалить Pod (руками или через Deployment уменьшив количество replicas дo 0).
- Pod размещается на определенном узле кластера Kubernetes
- Для Pod задается политика автоматического рестарта в рамках одного Node [restartPolicy](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#restart-policy): Always, OnFailure, Never.
- Остановкой и запуском Pod управляет Kubernetes. В общем случае, Pod может быть автоматически остановлен если:
  - все контейнеры в Pod завершили работу (успешно или ошибочно)
  - нарушены liveness probe для контейнера
  - нарушены limits.memory для контейнера
  - Pod больше не нужен, например, если уменьшилась нагрузка и Horizontal Autoscaler уменьшил количество replicas 
- При [остановке Pod](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-termination):
  - в контейнеры будет отправлен STOPSIGNAL
  - по прошествии terminationGracePeriodSeconds процессы контейнеров будут удалены
  - при удалении Pod и следующем создании, он может быть создан на другом узле кластера, с другими IP адресами и портами и к нему могут быть примонтированы новые Volume

При разработке stateless приложений для Kubernetes нужно учитывать эту специфику:
- приложение может быть остановлено в любой момент
  - необходимо предусмотреть возможность "чистой" остановки в течении grace period (закрыть подключения к БД, завершить (или нет) текущие задачи, закрыть HTTP соединения, ...)
  - необходимо предусмотреть возможность "жесткой" остановки, если приложение взаимодействие со stateful сервисами, то предусмотреть компенсирующие воздействия при повторном запуске
  - желательно контролировать лимит по памяти для приложения 
- при запуске/перезапуске приложения все данные в локальной файловой системе контейнера потенциально могут быть потеряны
- при запуске/перезапуске приложения данные на примонтированных Volume потенциально могут быть так же потеряны
- при остановке приложения доступ к stdout и stderr получить можно, но есть ограничения - критические логи желательно сразу отправлять во внешний сервис или на Volume, который точно не будет удален  
- в Kubernetes нет возможности явно задать последовательность старта различных Pod
  - приложение может потенциально запущено раньше других связанных сервисов 
  - желательно предусмотреть вариант постоянного (или лимитированного по количеству раз) перезапуска приложения в ожидании готовности внешних сервисов
  - желательно предусмотреть в связанных приложениях liveness/readiness probe, чтобы понять, когда связанное приложение готово к работе

## 8. Подготовка YAML для Kubernetes

### Конфигурирование
В Kubernetes предусмотрено для основных способа конфигурирования [ConfigMaps](https://kubernetes.io/docs/concepts/configuration/configmap/) для основных настроек и [Secrets](https://kubernetes.io/docs/concepts/configuration/secret/) для настроек, чувствительных с точки зрения безопасности.

В простейшем случае [/deploy/kubernates/base/app-configmap.yaml](https://github.com/romapres2010/goapp/blob/master/deploy/kubernates/base/app-configmap.yaml), содержит переменные со строковыми значениями. В более сложных вариантах, можно считывать и разбирать конфиги из внешнего файла или использовать внешний файл без "разбора".

Все артефакты Kubernetes желательно сразу создавать в отдельном namespace [/deploy/kubernates/base/app-namespase.yaml](https://github.com/romapres2010/goapp/blob/master/deploy/kubernates/base/app-namespase.yaml).

Для каждого артефакта нужно указывать метки для дальнейшего поиска и фильтрации. В примере приведена одна метка с именем 'app' и значением 'app'.


``` yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
  labels:
    app: app
  namespace: go-app
data:
  APP_CONFIG_FILE: /app/defcfg/app.global.yaml
  APP_HTTP_PORT: "8080"
  APP_HTTP_LISTEN_SPEC: 0.0.0.0:8080
  APP_LOG_LEVEL: ERROR
  APP_LOG_FILE: /app/log/app.log
  APP_PG_HOST: dev-app-db
  APP_PG_PORT: "5432"
  APP_PG_DBNAME: postgres
  APP_PG_CHANGELOG: db.changelog-1.0_recreate_testdatamdg.xml	
```
"Секретные" конфигурационные данные определяются в [/deploy/kubernates/base/app-secret.yaml](https://github.com/romapres2010/goapp/blob/master/deploy/kubernates/base/app-secret.yaml).
Данный вариант максимально упрощенный - правильно использовать внешнее зашифрованное хранилище. 

``` yaml
apiVersion: v1
kind: Secret
metadata:
  name: app-secret
  labels:
    app: app
  namespace: go-app
type: Opaque
stringData:
  APP_PG_USER: postgres
  APP_PG_PASS: postgres
```

### БД
Развертывание БД в шаблоне приведено в упрощенном stateless варианте (может быть использовано только для dev). 
Правильный вариант для промышленной эксплуатации - развертывание через оператор с поддержкой кластеризации, например, [zalando](https://github.com/zalando/postgres-operator).

Прежде всего, нужно запросить ресурсы для [Persistent Volumes](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) на котором будет располагаться файлы с БД.

Если PersistentVolumeClaim удаляется, то выделенный Persistent Volumes в зависимости от persistentVolumeReclaimPolicy, будет удален, очищен или сохранен.

_Хорошая статья с описанием [хранилищ данных (Persistent Volumes) в Kubernetes](https://serveradmin.ru/hranilishha-dannyh-persistent-volumes-v-kubernetes/)_

``` yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
    name: app-db-claim
    labels:
        app: app
    namespace: go-app
spec:
    accessModes:
        - ReadWriteOnce
    resources:
        requests:
            storage: 100Mi
```

Управлять созданием Pod удобнее через [Deployments](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/), который задает шаблон по которому будут создаваться Pod.
Особенности развертывания [/deploy/kubernates/base/app-db-deployment.yaml](https://github.com/romapres2010/goapp/blob/master/deploy/kubernates/base/app-db-deployment.yaml):
- количество replicas: 1 - создавать несколько экземпляров БД таким образом можно, но физически у каждой БД будут свои файлы.
- template.metadata.labels задана дополнительная метка tier: app-db, чтобы можно было легко найти Pod БД
- ENV переменные заполняются из ранее созданных ConfigMap и Secret
- определены readinessProbe и livenessProbe
- resources.limits для БД не указываются
- к /var/lib/postgresql/data примонтирован volume, полученный через ранее определенный  persistentVolumeClaim.claimName: app-db-claim

``` yaml
apiVersion: apps/v1
kind: Deployment
metadata:
    name: app-db
    labels:
        tier: app-db
    namespace: go-app
spec:
    replicas: 1
    selector:
        matchLabels:
            tier: app-db
    strategy:
        type: Recreate
    template:
        metadata:
            labels:
                app_net: "true"
                tier: app-db
        spec:
            containers:
                - name: app-db
                  env:
                    - name: POSTGRES_PASSWORD
                      valueFrom:
                        secretKeyRef:
                          name: app-secret
                          key: APP_PG_PASS
                    - name: POSTGRES_DB
                      valueFrom:
                        configMapKeyRef:
                          name: app-config
                          key: APP_PG_DBNAME
                    - name: POSTGRES_USER
                      valueFrom:
                        secretKeyRef:
                          name: app-secret
                          key: APP_PG_USER
                    - name: PGUSER
                      valueFrom:
                        secretKeyRef:
                          name: app-secret
                          key: APP_PG_USER
                  image: postgres:14.5-alpine
                  imagePullPolicy: IfNotPresent
                  readinessProbe:
                    exec:
                      command:
                        - pg_isready
                    initialDelaySeconds: 30  # Time to create a new DB
                    failureThreshold: 5
                    periodSeconds: 10
                    timeoutSeconds: 5
                  livenessProbe:
                    exec:
                      command:
                        - pg_isready
                    failureThreshold: 5
                    periodSeconds: 10
                    timeoutSeconds: 5
                  ports:
                    - containerPort: 5432
                  volumeMounts:
                    - mountPath: /var/lib/postgresql/data
                      name: app-db-volume
            hostname: app-db-host
            restartPolicy: Always
            volumes:
              - name: app-db-volume
                persistentVolumeClaim:
                  claimName: app-db-claim
```

Если доступ к БД нужен вне кластера, то можно определить [Service](https://kubernetes.io/docs/concepts/services-networking/service/) с типом NodePort - [/deploy/kubernates/base/app-db-service.yaml](https://github.com/romapres2010/goapp/blob/master/deploy/kubernates/base/app-db-service.yaml). 
- в этом случае на каждом узле кластера, где развернут Pod с БД будет открыт "рандомный" порт, на который будет смаплен порт 5432 из Docker контейнера с БД
- по этому "рандомному" порту и IP адресу узла кластера можно получить доступ к БД.
- назначить БД конкретный порт нельзя. При пересоздании Pod через stateless Deployment, порт может быть уже другим.
- для того, чтобы Service был смаплен с созданным через Deployment Pod БД, должны соответствовать Service.spec.selector.tier: app-db и Deployment.spec.template.metadata.labels.tier: app-db

``` yaml
apiVersion: v1
kind: Service
metadata:
    name: app-db
    labels:
        tier: app-db
    namespace: go-app
spec:
    type: NodePort   # A port is opened on each node in your cluster via Kube proxy.
    ports:
        - port: 5432
          targetPort: 5432
    selector:
        tier: app-db
```

### Liquibase
Liquibase со скриптами должен запускаться только один раз, сразу после старта БД: 
- нужно задержать запуск, пока БД не поднимется
- второй раз Liquibase не должен запускаться 
- нужно различать запуски для первичной инсталляции и установки изменений DDL и DML на существующую БД.

В простом случае, без использования Helm, запуск реализован в виде однократно запускаемого Pod - [/deploy/kubernates/base/app-liquibase-pod.yaml](https://github.com/romapres2010/goapp/blob/master/deploy/kubernates/base/app-liquibase-pod.yaml)
- ENV переменные заполняются из ранее созданных ConfigMap и Secret
- template.metadata.labels задана дополнительная метка tier: app-liquibase, чтобы можно было легко найти Pod Liquibase
- задан отдельный initContainers с image: busybox:1.28, задача которой - бесконечный цикл (лучше его ограничить, иначе придется руками удалять Pod) - ожидания готовности порта 5432 postgres в Pod c БД.
  - для прослушивания порта используется команда nc -w
  - здесь нам пригодился созданный ранее [Service]((https://github.com/romapres2010/goapp/blob/master/deploy/kubernates/base/app-db-service.yaml)) для БД. 
    - В рамках Kubernetes кластера в качестве краткого DNS имени используется именно Service.metadata.name, которое мы определили как 'app-db'. 
    - Это же значения мы записали в ConfigMap.data.APP_PG_HOST
    - ConfigMap.data.APP_PG_HOST смаплен на ENV переменную initContainers.env.APP_PG_HOST
    - ENV переменную initContainers.env.APP_PG_HOST используем в команде nc -w
- в command определена собственно строка запуска Liquibase с передачей через ENV переменную initContainers.env.APP_PG_CHANGELOG корневого скрипта для запуска '--changelog-file=./changelog/$(APP_PG_CHANGELOG)'
- чтобы исключить повторный запуск Liquibase Pod указано spec.restartPolicy: Never
- tag для Docker образа будет определен в kustomize через подстановку image: app-liquibase 

``` yaml
apiVersion: v1
kind: Pod
metadata:
    name: app-liquibase
    labels:
        tier: app-liquibase
    namespace: go-app
spec:
    initContainers:
        - name: init-app-liquibase
          env:
            - name: APP_PG_HOST
              valueFrom:
                configMapKeyRef:
                  name: app-config
                  key: APP_PG_HOST
            - name: APP_PG_PORT
              valueFrom:
                configMapKeyRef:
                  name: app-config
                  key: APP_PG_PORT
          image: busybox:1.28
          command: ['sh', '-c', "until nc -w 2 $(APP_PG_HOST) $(APP_PG_PORT); do echo Waiting for $(APP_PG_HOST):$(APP_PG_PORT) to be ready; sleep 5; done"]
    containers:
        - name: app-liquibase
          env:
            - name: APP_PG_USER
              valueFrom:
                secretKeyRef:
                  name: app-secret
                  key: APP_PG_USER
            - name: APP_PG_PASS
              valueFrom:
                secretKeyRef:
                  name: app-secret
                  key: APP_PG_PASS
            - name: APP_PG_HOST
              valueFrom:
                configMapKeyRef:
                  name: app-config
                  key: APP_PG_HOST
            - name: APP_PG_PORT
              valueFrom:
                configMapKeyRef:
                  name: app-config
                  key: APP_PG_PORT
            - name: APP_PG_DBNAME
              valueFrom:
                configMapKeyRef:
                  name: app-config
                  key: APP_PG_DBNAME
            - name: APP_PG_CHANGELOG
              valueFrom:
                configMapKeyRef:
                  name: app-config
                  key: APP_PG_CHANGELOG
          image: app-liquibase # image: romapres2010/app-liquibase:2.0.0
          command: ['sh', '-c', "docker-entrypoint.sh --changelog-file=./changelog/$(APP_PG_CHANGELOG) --url=jdbc:postgresql://$(APP_PG_HOST):$(APP_PG_PORT)/$(APP_PG_DBNAME) --username=$(APP_PG_USER) --password=$(APP_PG_PASS) --logLevel=info update"]
          imagePullPolicy: IfNotPresent
    restartPolicy: Never
```

### Go Api

Особенности развертывания [/deploy/kubernates/base/app-api-deployment.yaml](https://github.com/romapres2010/goapp/blob/master/deploy/kubernates/base/app-api-deployment.yaml):
- начальное количество replicas: 1, остальные будет автоматически создаваться Horizontal Autoscaler
- template.metadata.labels задана дополнительная метка tier: app-api, чтобы можно было легко найти Pod Go App
- ENV переменные заполняются из ранее созданных ConfigMap и Secret
- по аналогии с Liquibase задан отдельный initContainers для ожидания готовности порта 5432 postgres в Pod c БД
  - так как все Pod (БД, Liquibase, Go Api) будут запущены одновременно, то возникнет ситуация, когда БД уже стартовала, но Liquibase еще не применил DDL и DML скрипты. Или применил их только частично. 
  - в это время контейнер с Go Api (в текущем шаблоне) будет падать с ошибкой и пересоздаваться.
  - поэтому нужно правильно настроить readinessProbe для Go Api, которая определяет с какого момента приложение готово к работе
  - альтернативный вариант иметь в БД метку или сигнал, об успешной установке патчей на БД по которому Go Api будет готова к работе    
- определены readinessProbe и livenessProbe на основе httpGet
- определена период мягкой остановки - terminationGracePeriodSeconds: 45
- заданы resources.requests - это определяет минимальные ресурсы, для старта приложения. В зависимости от этого будет выбран узел кластера со свободными ресурсами.
- заданы resources.limits
  - если превышен limits.memory, то Pod будет удален и пересоздан
  - limits.cpu контролируется собственно кластером - больше процессорного времени не будет выделено. 
- tag для Docker образа будет определен в kustomize через подстановку image: app-api

``` yaml
apiVersion: apps/v1
kind: Deployment
metadata:
    name: app-api
    labels:
        tier: app-api
    namespace: go-app
spec:
    replicas: 1
    selector:
        matchLabels:
            tier: app-api
    strategy:
        type: Recreate
    template:
        metadata:
            labels:
                app_net: "true"
                tier: app-api
        spec:
            initContainers:
                - name: init-app-api
                  env:
                    - name: APP_PG_HOST
                      valueFrom:
                        configMapKeyRef:
                          name: app-config
                          key: APP_PG_HOST
                    - name: APP_PG_PORT
                      valueFrom:
                        configMapKeyRef:
                          name: app-config
                          key: APP_PG_PORT
                  image: busybox:1.28
                  command: ['sh', '-c', "until nc -w 2 $(APP_PG_HOST) $(APP_PG_PORT); do echo Waiting for $(APP_PG_HOST):$(APP_PG_PORT) to be ready; sleep 5; done"]
            terminationGracePeriodSeconds: 45
            containers:
                - name: app-api
                  env:
                    - name: APP_CONFIG_FILE
                      valueFrom:
                        configMapKeyRef:
                          name: app-config
                          key: APP_CONFIG_FILE
                    - name: APP_HTTP_LISTEN_SPEC
                      valueFrom:
                        configMapKeyRef:
                          name: app-config
                          key: APP_HTTP_LISTEN_SPEC
                    - name: APP_LOG_LEVEL
                      valueFrom:
                        configMapKeyRef:
                          name: app-config
                          key: APP_LOG_LEVEL
                    - name: APP_LOG_FILE
                      valueFrom:
                        configMapKeyRef:
                          name: app-config
                          key: APP_LOG_FILE
                    - name: APP_PG_USER
                      valueFrom:
                        secretKeyRef:
                          name: app-secret
                          key: APP_PG_USER
                    - name: APP_PG_PASS
                      valueFrom:
                        secretKeyRef:
                          name: app-secret
                          key: APP_PG_PASS
                    - name: APP_PG_HOST
                      valueFrom:
                        configMapKeyRef:
                          name: app-config
                          key: APP_PG_HOST
                    - name: APP_PG_PORT
                      valueFrom:
                        configMapKeyRef:
                          name: app-config
                          key: APP_PG_PORT
                    - name: APP_PG_DBNAME
                      valueFrom:
                        configMapKeyRef:
                          name: app-config
                          key: APP_PG_DBNAME
                  image: app-api # image: romapres2010/app-api:2.0.0
                  imagePullPolicy: IfNotPresent
                  readinessProbe:
                    httpGet:
                      path: /app/system/health
                      port: 8080
                      scheme: HTTP
                    initialDelaySeconds: 30  # Time to start
                    failureThreshold: 5
                    periodSeconds: 10
                    timeoutSeconds: 5
                  livenessProbe:
                    httpGet:
                      path: /app/system/health
                      port: 8080
                      scheme: HTTP
                    failureThreshold: 5
                    periodSeconds: 10
                    timeoutSeconds: 5
                  ports:
                    - containerPort: 8080
                  resources:
                    requests:
                      cpu: 500m
                      memory: 256Mi
                    limits:
                      cpu: 2000m
                      memory: 2000Mi
            hostname: app-api-host
            restartPolicy: Always
```

Доступ к Go Api нужен вне кластера, причем так как Pod может быть несколько, то нужен LoadBalancer - [/deploy/kubernates/base/app-api-service.yaml](https://github.com/romapres2010/goapp/blob/master/deploy/kubernates/base/app-api-service.yaml).
- в нашем случае для кластера настраивается внешний порт 3000 на IP адрес собственно кластера
- для того, чтобы Service был смаплен с созданным через Deployment Pod Go APi (при масштабировании нагрузки их будет несколько), должны соответствовать Service.spec.selector.tier: app-api и Deployment.spec.template.metadata.labels.tier: app-api

``` yaml
apiVersion: v1
kind: Service
metadata:
    name: app-api
    labels:
        tier: app-api
    namespace: go-app
spec:
    type: LoadBalancer
    ports:
        - port: 3000
          targetPort: 8080
    selector:
        tier: app-api
```

## 9. Kustomization YAML для Kubernetes

Kubernetes не предоставляет стандартной возможности использовать внешние ENV переменные - каждый раз нужно менять YAML и заново "накатывать конфигурацию".

Самый простой автоматически вносить изменения в YAML файлы в зависимости от сред развертывания DEV-TEST-PROD - это [Kustomize](https://kustomize.io/).
- создается "базовая версия" (не содержит специфику сред) YAML для Kubernetes - она выложена в каталоге [/deploy/kubernates/base](https://github.com/romapres2010/goapp/tree/master/deploy/kubernates/base)
- для каждой из сред (вариантов развертывания) создает отдельный каталог, содержащий изменения в YAML, которые должны быть применены поверх "базовой версии". Например, [/deploy/kubernates/overlays/dev](https://github.com/romapres2010/goapp/tree/master/deploy/kubernates/overlays/dev). 

Состав YAML "базовой версии" определен в [/deploy/kubernates/base/kustomization.yaml](https://github.com/romapres2010/goapp/blob/master/deploy/kubernates/base/kustomization.yaml). 

``` yaml
commonLabels:
  app: app
  variant: base
resources:
- app-namespase.yaml
- app-net-networkpolicy.yaml
- app-configmap.yaml
- app-secret.yaml
- app-db-persistentvolumeclaim.yaml
- app-db-deployment.yaml
- app-db-service.yaml
- app-api-deployment.yaml
- app-api-service.yaml
- app-liquibase-pod.yaml
```

Состав изменений, которые нужно применить для среды DEV к "базовой версии" определен в [/deploy/kubernates/overlays/dev/kustomization.yaml](https://github.com/romapres2010/goapp/blob/master/deploy/kubernates/overlays/dev/kustomization.yaml).
- в имена всех артефактов Kubernetes добавлен дополнительный префикс namePrefix: dev- 
- определена новая метка, для фильтрации всех артефактов среды - commonLabels.variant: dev
- определены подстановки реальных Docker образов для app-api и app-liquibase
- определены патчи, которые нужно применить поверх "базовой версии"

``` yaml
namePrefix: dev-
commonLabels:
  variant: dev
commonAnnotations:
  note: This is development
resources:
- ../../base
images:
- name: app-api
  newName: romapres2010/app-api
  newTag: 2.0.0
- name: app-liquibase
  newName: romapres2010/app-liquibase
  newTag: 2.0.0
patches:
- app-configmap.yaml
- app-secret.yaml
- app-api-deployment.yaml
```

### Патчи поверх базовой версии
Включают изменения, специфичные для сред: IP, порты, имена схем БД, пароли, ресурсы, лимиты.

Изменение Secret
``` yaml
apiVersion: v1
kind: Secret
metadata:
  name: app-secret
  labels:
    app: app
  namespace: go-app
type: Opaque
stringData:
  APP_PG_USER: postgres
  APP_PG_PASS: postgres
```

Изменение ConfigMap
``` yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
  labels:
    app: app
  namespace: go-app
data:
  APP_CONFIG_FILE: /app/defcfg/app.global.yaml
  APP_HTTP_PORT: "8080"
  APP_HTTP_LISTEN_SPEC: 0.0.0.0:8080
  APP_LOG_LEVEL: ERROR
  APP_LOG_FILE: /app/log/app.log
  APP_PG_HOST: dev-app-db
  APP_PG_PORT: "5432"
  APP_PG_DBNAME: postgres
  APP_PG_CHANGELOG: db.changelog-1.0_recreate_testdatamdg.xml	
```

Изменение Deployment Go Api - заданы другие лимиты, количество реплик и время "чистой" остановки
``` yaml
apiVersion: apps/v1
kind: Deployment
metadata:
    name: app-api
    labels:
        tier: app-api
    namespace: go-app
spec:
    replicas: 2
    template:
        spec:
            terminationGracePeriodSeconds: 60
            containers:
                - name: app-api
                  resources:
                      limits:
                          cpu: 4000m
                          memory: 4000Mi
            restartPolicy: Always
```

## 10. Тестирование Kubernetes с kustomize
Для тестирования подготовлено несколько скриптов в каталоге [/deploy/kubernates](https://github.com/romapres2010/goapp/tree/master/deploy/kubernates).

Для тестирования под Windows достаточно поставить Docker Desktop и включить в нем опцию Enable Kubernetes.

_скрипты представлены только для ознакомительных целей_

### Сборка Docker образов

В файле [/deploy/default_repository](https://github.com/romapres2010/goapp/blob/master/deploy/default_repository) нужно подменить константу на свой Docker репозиторий. Без этого тоже будет работать, но не получится сделать docker push.  

Собрать Docker образы:
- Go App [/deploy/docker/app-api-build.sh](https://github.com/romapres2010/goapp/blob/master/deploy/docker/app-api-build.sh)
- Liquibase [/deploy/docker/app-liquibase-build.sh](https://github.com/romapres2010/goapp/blob/master/deploy/docker/app-liquibase-build.sh)

Выполнять docker push не обязательно, так как после сборки Docker образы кешируются на локальной машине.   

### Развертывание артефактов Kubernetes

Запускаем скрипт [/deploy/kubernates/kube-build.sh](https://github.com/romapres2010/goapp/blob/master/deploy/kubernates/kube-build.sh) и передаем ему в качестве параметров среду dev, которую нужно развернуть.

``` shell
./kube-build.sh dev
```

Все результаты логируются в каталог [/deploy/kubernates/log](https://github.com/romapres2010/goapp/tree/master/deploy/kubernates/log)
- первым шагом выполняется kubectl kustomize и формируется итоговый YAML для среды DEV [/kubernates/log/build-dev-kustomize.yaml](https://github.com/romapres2010/goapp/blob/master/deploy/kubernates/log/build-dev-kustomize.yaml) - можно посмотреть как применилась dev конфигурация
- вторым шагом выполняется kubectl apply - запускается одновременное создание всех ресурсов.
- стартуют сразу все Pod, но dev-app-api и dev-app-liquibase уходит в ожидание через initContainers готовности порта 5432 postgres в Pod c БД
- вставлена задержка 120 сек. и после этого собираются логи всех контейнеров

### Краткий статус артефактов Kubernetes

Если выполнить скрипт [/master/deploy/kubernates/kube-descibe.sh](https://github.com/romapres2010/goapp/blob/master/deploy/kubernates/kube-descibe.sh), то можно посмотреть краткий статус артефактов Kubernetes.
- Pod Liquibase успешно отработал и остановился. Его создавали не через Deployment, поэтому у него "нормальное имя" _dev-app-liquibase_.  
- Запущены и работают два Pod Go Api, они создавались через Deployment, поэтому имена имеют "нормальный" префикс и автоматически сгенерированную часть.
- Запущен и работает Pod БД.
- Сервис dev-app-api имеет тип LoadBalancer
  - он доступен вне кластера на 3000 порту - например, можно вызвать [/metrics](http://127.0.0.1:3000/metrics).
  - стандартный LoadBalancer работает по алгоритму [round robin](https://ru.wikipedia.org/wiki/Round-robin_(%D0%B0%D0%BB%D0%B3%D0%BE%D1%80%D0%B8%D1%82%D0%BC)), поэтому без сохранения сессии запросы будут направляться на разные Pod по очереди.
  - если клиент запросит HTTP keep-alive, то запросы будут идти на тот же Pod (это имеет значение при использовании Horizontal Autoscaler) 
- Сервис dev-app-db имеет тип NodePort и указан назначенный ему порт 30906
  - что бы получить досут к БД вне кластера, нужно сначала определить на каком узле (Node) кластера поднят этот Pod и по IP адресу узла и порту 30906 можно получить доступ к БД
- Между собой Pod могут коммуницировать через краткое DNS имя service.
  - Liquibase и Go Api могут обратиться к БД через host name = service: dev-app-db

``` shell
$ ./kube-get.sh dev go-app

Kube namespace: go-app
Kube variant: dev

kubectl get pods
NAME                           READY   STATUS      RESTARTS   AGE   IP           NODE             NOMINATED NODE   READINESS GATES
dev-app-api-59b6ff97b4-2kmfm   1/1     Running     0          13m   10.1.1.182   docker-desktop   <none>           <none>
dev-app-api-59b6ff97b4-plmz6   1/1     Running     0          13m   10.1.1.183   docker-desktop   <none>           <none>
dev-app-db-58bbb867d8-c96bz    1/1     Running     0          13m   10.1.1.184   docker-desktop   <none>           <none>
dev-app-liquibase              0/1     Completed   0          13m   10.1.1.185   docker-desktop   <none>           <none>

kubectl get deployment
NAME          READY   UP-TO-DATE   AVAILABLE   AGE   CONTAINERS   IMAGES                       SELECTOR
dev-app-api   2/2     2            2           13m   app-api      romapres2010/app-api:2.0.0   app=app,tier=app-api,variant=dev
dev-app-db    1/1     1            1           13m   app-db       postgres:14.5-alpine         app=app,tier=app-db,variant=dev

kubectl get service
NAME          TYPE           CLUSTER-IP       EXTERNAL-IP   PORT(S)          AGE   SELECTOR
dev-app-api   LoadBalancer   10.106.114.183   localhost     3000:30894/TCP   13m   app=app,tier=app-api,variant=dev
dev-app-db    NodePort       10.111.201.17    <none>        5432:30906/TCP   13m   app=app,tier=app-db,variant=dev

kubectl get configmap
NAME             DATA   AGE
dev-app-config   9      13m

kubectl get secret
NAME             TYPE     DATA   AGE
dev-app-secret   Opaque   2      13m

kubectl get pvc
NAME               STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE   VOLUMEMODE
dev-app-db-claim   Bound    pvc-996244d5-c5fd-4496-abfd-b6d9301549af   100Mi      RWO            hostpath       13m   Filesystem

kubectl get hpa
No resources found in go-app namespace.
```

### Развернутый статус артефактов Kubernetes

Развернутый статус артефактов Kubernetes можно посмотреть, запустив скрипт [/master/deploy/kubernates/kube-descibe.sh](https://github.com/romapres2010/goapp/blob/master/deploy/kubernates/kube-descibe.sh).

``` shell
$ ./kube-descibe.sh dev go-app
```

Пример результатов в файле [/deploy/kubernates/log/describe-dev-kube.log](https://github.com/romapres2010/goapp/blob/master/deploy/kubernates/log/describe-dev-kube.log).

### Проверка логов Docker контейнеров

Логи контейнеров Kubernetes можно посмотреть, запустив скрипт [/deploy/kubernates/kube-log.sh](https://github.com/romapres2010/goapp/blob/master/deploy/kubernates/kube-log.sh).

``` shell
$ ./kube-log.sh dev go-app
```

У нас запущено 2 Pod Go Api - в этом скрипте они будут перемешаны и записаны в один файл.  


### Удаление артефактов Kubernetes

Удалить все артефакты можно скриптом [/deploy/kubernates/kube-delete.sh](https://github.com/romapres2010/goapp/blob/master/deploy/kubernates/kube-delete.sh).

``` shell
$ ./kube-deelte.sh dev go-app
```
