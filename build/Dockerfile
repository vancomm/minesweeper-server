# syntax=docker/dockerfile:1

ARG GO_VERSION=1.23.2

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
    go build -o ./bin/gateway ./cmd/gateway

RUN --mount=type=cache,target=/go/pkg/mod/ \
    CGO_ENABLED=$CGO_ENABLED \
    GOARCH=$TARGETARCH \
    go build -o ./bin/game ./cmd/game

FROM debian:latest AS runtime

WORKDIR /app

ARG UID=10001

RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    appuser
USER appuser

COPY --from=build \
    /app/bin /app/bin

EXPOSE 8080

FROM runtime AS gateway

ENTRYPOINT [ "/app/bin/gateway" ]

FROM runtime AS game

ENTRYPOINT [ "/app/bin/game" ]

FROM runtime AS migrator

ENTRYPOINT [ "/app/bin/migrator" ]