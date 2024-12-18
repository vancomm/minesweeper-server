SOURCES_DIR 			= cmd
BINARY_DIR				= bin
CGO_ENABLED 			= 1

.PHONY: all
all: gateway game

gateway: $(wildcard ${SOURCES_DIR}/gateway/*.go)
	go build -o ${BINARY_DIR}/gateway $@

game: $(wildcard ${SOURCES_DIR}/game/*.go)
	go build -o ${BINARY_DIR}/game $@

.PHONY: test
test:
	go test -v ./...

.PHONY: clean
clean:
	go clean
	rm ${BINARY_NAME}