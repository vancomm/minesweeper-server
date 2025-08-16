# syntax=docker/dockerfile:1

ARG GO_VERSION=1.24.0

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS build

WORKDIR /app

COPY go.mod go.sum /app/

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x

COPY . /app/

ARG CGO_ENABLED=1

ARG TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod/ \
    CGO_ENABLED=$CGO_ENABLED \
    GOARCH=$TARGETARCH \
    go build -o ./bin/server ./cmd/server \
    && go build -o ./bin/migrator ./cmd/migrator

FROM debian:13 AS runtime

WORKDIR /app

ARG OS_GROUP_ID=42000
ARG OS_GROUP_NAME=minesapp
ARG OS_UID=10000
ARG OS_USER_NAME=minesapp

RUN groupadd \
    --gid "${OS_GROUP_ID}" \
    "${OS_GROUP_NAME}"

RUN useradd \
    --gid "${OS_GROUP_ID}" \
    --shell "/sbin/nologin" \
    "${OS_USER_NAME}"

USER "${OS_USER_NAME}"

COPY --from=build \
    /app/bin /app/bin

EXPOSE 8000

FROM runtime AS server

ENTRYPOINT [ "/app/bin/server" ]

FROM runtime AS migrator

ENTRYPOINT [ "/app/bin/migrator" ]