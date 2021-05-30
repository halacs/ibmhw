all: lint build test

lint:
	# GO111MODULE=on go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.26.0
	golangci-lint run

build:
	go build

test:
	go test --cover
