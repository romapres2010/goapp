##
## Build stage
##

FROM golang:1.19-buster AS build

ARG APP_COMMIT
ARG APP_BUILD_TIME
ARG APP_VERSION

WORKDIR /app

RUN mkdir ./run && mkdir ./run/defcfg && mkdir ./run/log  # default directory

COPY ./vendor ./vendor
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
COPY ./pkg ./pkg
COPY ./cmd/app/main.go ./cmd/app/main.go
COPY ./deploy/config/. ./run/defcfg/

RUN go build -v -mod vendor -ldflags "-X main.commit=${APP_COMMIT} -X main.buildTime=${APP_BUILD_TIME} -X main.version=${APP_VERSION}" -o ./run/main ./cmd/app/main.go

RUN echo "Based on commit: $APP_COMMIT" && echo "Build Time: $APP_BUILD_TIME" && echo "Version: $APP_VERSION"

##
## Deploy stage
##
FROM gcr.io/distroless/base-debian10

WORKDIR /app

COPY --from=build /app/run/. .

EXPOSE 8080/tcp

ENTRYPOINT [ "/app/main"]
