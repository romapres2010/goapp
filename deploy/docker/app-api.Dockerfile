##
## Build stage
##

FROM golang:1.19-buster AS build

# Git commit, время сборки и базовая версия приложения передаются через аргументы docker
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
