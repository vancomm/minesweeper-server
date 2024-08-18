SOURCES 	= $(wildcard *.go) $(wildcard */*.go)
BINARY_NAME = bin/main.out
CGO_ENABLED = 1

.PHONY: all
all: ${BINARY_NAME}

${BINARY_NAME}: $(SOURCES)
	go build -o $@ *.go

.PHONY: run
run: ${BINARY_NAME}
	@$<
 
.PHONY: test
test:
	go test -v ./...

.PHONY: clean
clean:
	go clean
	rm ${BINARY_NAME}