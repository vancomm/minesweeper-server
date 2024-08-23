SOURCES 			= $(wildcard *.go) $(wildcard */*.go)
BINARY_NAME 		= bin/main.out
CGO_ENABLED 		= 1
LOCAL_COMPOSE_FLAGS	= -f compose.yml -f compose.local.yml

.PHONY: all
all: ${BINARY_NAME}

${BINARY_NAME}: $(SOURCES)
	go build -o $@ *.go

.PHONY: dev
dev: ${BINARY_NAME}
	docker compose ${LOCAL_COMPOSE_FLAGS} up --build
# dev: ${BINARY_NAME}
# 	@$<

.PHONY: test
test:
	go test -v ./...

.PHONY: clean
clean:
	go clean
	rm ${BINARY_NAME}