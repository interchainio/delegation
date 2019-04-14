OUTPUTDIR ?= build
GOPATH ?= $(shell go env GOPATH)

.PHONY: build test lint package

build:
		GO111MODULE=on go build -o $(OUTPUTDIR)/delegation cmd/delegation/main.go
		GO111MODULE=on go build -o $(OUTPUTDIR)/stake-dist cmd/stake-dist/main.go

test:
		GO111MODULE=on go test -cover -race ./...

$(GOPATH)/bin/golangci-lint:
		GO111MODULE=off go get -u github.com/golangci/golangci-lint/cmd/golangci-lint

lint: $(GOPATH)/bin/golangci-lint
		GO111MODULE=on $(GOPATH)/bin/golangci-lint run ./...
