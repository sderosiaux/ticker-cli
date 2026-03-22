.PHONY: build test lint install clean

build:
	go build -o ticker-cli .

test:
	go test ./... -count=1 -race

lint:
	golangci-lint run ./...

install:
	go install .

clean:
	rm -f ticker-cli
