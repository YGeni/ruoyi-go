# server-go 常用命令（1a 期）
# 工作目录：server-go/

GO ?= go
BIN := bin/server
CFG := config/config.yaml

.PHONY: build vet test run clean

build:
	$(GO) build -o $(BIN) ./cmd/server

vet:
	$(GO) vet ./...

test:
	$(GO) test ./... -count=1

run:
	$(GO) run ./cmd/server -c $(CFG)

clean:
	rm -rf bin
