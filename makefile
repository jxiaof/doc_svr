APP_NAME ?= doc-svr
PORT ?= 4000
GOFLAGS ?= -mod=vendor
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS ?= -s -w -X main.version=$(VERSION)
DOCKER_IMAGE ?= $(APP_NAME):$(VERSION)

.PHONY: help fmt tidy vendor generate-home regen build run serve check smoke clean docker-build

help:
	@printf "Targets:\n"
	@printf "  make fmt           # format go files\n"
	@printf "  make tidy          # sync go.mod/go.sum\n"
	@printf "  make vendor        # refresh vendor directory\n"
	@printf "  make generate-home # render homepage into public/index.html\n"
	@printf "  make regen         # alias of generate-home\n"
	@printf "  make build         # build ./bin/$(APP_NAME)\n"
	@printf "  make run           # run local server\n"
	@printf "  make serve         # alias of run\n"
	@printf "  make check         # compile validation\n"
	@printf "  make smoke         # generate homepage and run compile smoke check\n"
	@printf "  make clean         # remove generated artifacts\n"
	@printf "  make docker-build  # build container image\n"

fmt:
	@gofmt -w main.go $$(find internal -name '*.go' -type f)

tidy:
	@go mod tidy

vendor:
	@go mod vendor

generate-home:
	@GOFLAGS=$(GOFLAGS) go run -ldflags "$(LDFLAGS)" . generate-home --output public/index.html

regen: generate-home

build: fmt generate-home
	@mkdir -p bin
	@echo ">> building $(APP_NAME) ($(VERSION))"
	@CGO_ENABLED=0 GOFLAGS=$(GOFLAGS) go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(APP_NAME) .

run: generate-home
	@echo ">> running on :$(PORT)"
	@PORT=$(PORT) GOFLAGS=$(GOFLAGS) go run -ldflags "$(LDFLAGS)" .

serve: run

check:
	@GOFLAGS=$(GOFLAGS) go build ./...

smoke: generate-home check
	@grep -q 'registered-pages' public/index.html
	@printf ">> smoke passed: public/index.html regenerated and compile check succeeded\n"

clean:
	@rm -rf bin doc_svr
	@find . -name '.DS_Store' -delete

docker-build:
	@docker build --build-arg VERSION=$(VERSION) -t $(DOCKER_IMAGE) -f dockerfile .
