# server-go 常用命令（1a 期）
# 工作目录：server-go/

GO ?= go
BIN := bin/server
CFG := config/config.yaml

.PHONY: build vet test run dev clean

# 热重载。用 air 不是 fresh —— fresh 的构建写死成 `go build <root>`，
# 没法指定 main 包（本项目的 main 在 ./cmd/server），详见 .air.toml 的注释。
dev:
	air

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
