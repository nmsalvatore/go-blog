.PHONY: build build-linux run test fmt vet clean

build:
	go build -o blog .

build-linux:
	GOOS=linux GOARCH=amd64 go build -o blog .

run:
	go run .

test:
	go test ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	rm -f blog
