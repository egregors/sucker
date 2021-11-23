.PHONY: build clean test bench

all: run

run:
	pbpaste | ./sucker

build:
	GO111MODULE=on CGO_ENABLED=0 go build -mod=vendor -o sucker ./main.go

clean:
	rm -rf ./sucker_downloads

test:
	go test -v -count 1 -race -cover ./...

bench:
	go test -v -run Bench -bench=. ./...