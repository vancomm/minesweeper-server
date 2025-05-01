# syntax=docker/dockerfile:1

ARG GO_VERSION=1.24.0

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS build

WORKDIR /app

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x

COPY . /app
    
RUN --mount=type=cache,target=/go/pkg/mod \
    go install golang.org/x/tools/cmd/stringer

RUN go generate ./...

ARG CGO_ENABLED=1

ARG TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod/ \
    CGO_ENABLED=$CGO_ENABLED \
    GOARCH=$TARGETARCH \
    go build -o ./bin/migrator ./cmd/migrator

RUN --mount=type=cache,target=/go/pkg/mod/ \
    CGO_ENABLED=$CGO_ENABLED \
    GOARCH=$TARGETARCH \
    go build -o ./bin/server ./cmd/server

FROM debian:12.10-slim@sha256:b1211f6d19afd012477bd34fdcabb6b663d680e0f4b0537da6e6b0fd057a3ec3 AS runtime

WORKDIR /app

ARG OS_GROUP_ID=42000
ARG OS_GROUP_NAME=mines
ARG OS_UID=10000
ARG OS_USER_NAME=minesapp

RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${OS_UID}" \
    "${OS_USER_NAME}"   
RUN groupadd -g "${OS_GROUP_ID}" "${OS_GROUP_NAME}"
RUN usermod -aG "${OS_GROUP_NAME}" "${OS_USER_NAME}"

USER "${OS_USER_NAME}"

COPY --from=build \
    /app/bin /app/bin

EXPOSE 8080

FROM runtime AS server

ENTRYPOINT [ "/app/bin/server" ]

FROM runtime AS migrator

ENTRYPOINT [ "/app/bin/migrator" ]