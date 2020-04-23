OUTPUTDIR ?= build
GOPATH ?= $(shell go env GOPATH)

.PHONY: build test lint package

build:
		GO111MODULE=on CGO_ENABLED=0 go build -ldflags "-extldflags static" -o $(OUTPUTDIR)/delegation cmd/delegation/main.go
		GO111MODULE=on CGO_ENABLED=0 go build -ldflags "-extldflags static" -o $(OUTPUTDIR)/stake-dist cmd/stake-dist/main.go

test:
		GO111MODULE=on go test -cover -race ./...

$(GOPATH)/bin/golangci-lint:
		wget -O- -nv https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -x -d -b $$(go env GOPATH)/bin latest

lint: $(GOPATH)/bin/golangci-lint
		$(GOPATH)/bin/golangci-lint run ./...
