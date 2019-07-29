build:
	GO111MODULE=on CGO_ENABLED=0 go build -mod=vendor -o sucker ./app/cmd/main.go

test:
	go test -v -count 1 -race -cover ./...

bench:
	go test -v -run Bench -bench=. ./...