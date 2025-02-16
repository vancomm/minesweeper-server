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

FROM runtime AS server

ENTRYPOINT [ "/app/bin/server" ]

FROM runtime AS migrator

ENTRYPOINT [ "/app/bin/migrator" ]