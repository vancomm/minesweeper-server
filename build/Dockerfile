# syntax=docker/dockerfile:1

ARG GO_VERSION=1.23.2
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS build
WORKDIR /src

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x

ARG TARGETARCH

COPY . /src

RUN go install golang.org/x/tools/cmd/stringer

RUN go generate ./...

ARG SERVER_SOURCES

RUN --mount=type=cache,target=/go/pkg/mod/ \
    CGO_ENABLED=1 GOARCH=$TARGETARCH go build -o /bin/server ${SERVER_SOURCES}

FROM debian:latest AS final

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

COPY --from=build /bin/server /bin/

EXPOSE 8080

ENTRYPOINT [ "/bin/server" ]