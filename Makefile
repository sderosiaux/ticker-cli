.PHONY: build test install clean

build:
	go build -o ticker-cli .

test:
	go test ./... -count=1

install:
	go install .

clean:
	rm -f ticker-cli
