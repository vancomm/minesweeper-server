CGO_ENABLED=1
BINARY_NAME=bin/main.out

all: ${BINARY_NAME}

.PHONY: all

${BINARY_NAME}:
	go build -o $@ *.go
 
run: ${BINARY_NAME}
	@$<
 
test:
	go test -v ./...

clean:
	go clean
	rm ${BINARY_NAME}