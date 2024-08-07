CGO_ENABLED=1
BINARY_NAME=bin/main.out
 
build:
	go build -o ${BINARY_NAME} *.go
 
run:
	go build -o ${BINARY_NAME} *.go
	./${BINARY_NAME}
 
test:
	go test -v ./...

clean:
	go clean
	rm ${BINARY_NAME}