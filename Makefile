SOURCES 				= $(wildcard *.go) $(wildcard */*.go)
BINARY_NAME 			= bin/main.out
CGO_ENABLED 			= 1

LOCAL_COMPOSE_FLAGS		= -f compose.local.yml
STAGING_DOCKER_FLAGS	= -c remote
STAGING_COMPOSE_FLAGS	= -f compose.staging.yml

.PHONY: all
all: ${BINARY_NAME}

${BINARY_NAME}: $(SOURCES)
	go build -o $@ *.go

.PHONY: dev
dev: ${BINARY_NAME}
	docker compose ${LOCAL_COMPOSE_FLAGS} up --build

.PHONY: stage
stage: ${BINARY_NAME}
	docker ${STAGING_DOCKER_FLAGS} compose ${STAGING_COMPOSE_FLAGS} up --build

.PHONY: test
test:
	go test -v ./...

.PHONY: clean
clean:
	go clean
	rm ${BINARY_NAME}