.DEFAULT_GOAL := build
.PHONY: build build-linux run test fmt vet clean

fmt:
	go fmt ./...

vet:fmt
	go vet ./...

build:vet
	go build -o blog .

build-linux:
	GOOS=linux GOARCH=amd64 go build -o blog .

run:
	go run .

test:
	go test ./...

clean:
	go clean .
