APP_NAME ?= doc-svr
PORT ?= 3000
GOFLAGS ?= -mod=vendor
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS ?= -s -w -X main.version=$(VERSION)
DOCKER_IMAGE ?= $(APP_NAME):$(VERSION)

.PHONY: help fmt tidy vendor build run check clean docker-build

help:
	@printf "Targets:\n"
	@printf "  make fmt           # format go files\n"
	@printf "  make tidy          # sync go.mod/go.sum\n"
	@printf "  make vendor        # refresh vendor directory\n"
	@printf "  make build         # build ./bin/$(APP_NAME)\n"
	@printf "  make run           # run local server\n"
	@printf "  make check         # compile validation\n"
	@printf "  make docker-build  # build container image\n"

fmt:
	@gofmt -w main.go $$(find internal -name '*.go' -type f)

tidy:
	@go mod tidy

vendor:
	@go mod vendor

build: fmt
	@mkdir -p bin
	@echo ">> building $(APP_NAME) ($(VERSION))"
	@CGO_ENABLED=0 GOFLAGS=$(GOFLAGS) go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(APP_NAME) .

run:
	@echo ">> running on :$(PORT)"
	@PORT=$(PORT) GOFLAGS=$(GOFLAGS) go run -ldflags "$(LDFLAGS)" .

check:
	@GOFLAGS=$(GOFLAGS) go build ./...

clean:
	@rm -rf bin

docker-build:
	@docker build --build-arg VERSION=$(VERSION) -t $(DOCKER_IMAGE) -f dockerfile .
