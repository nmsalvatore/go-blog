.PHONY: build build-linux run test clean

build:
	go build -o blog .

build-linux:
	GOOS=linux GOARCH=amd64 go build -o blog .

run:
	go run .

test:
	go test ./...

clean:
	rm -f blog
