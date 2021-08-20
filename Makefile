BINDIR := bin
ROOT_PACKAGE := $(shell go list .)
VERSION := $(shell git describe --tags --abbrev=0)
REVISION := $(shell git rev-parse --short HEAD)
#GO_LDFLAGS_VERSION := -X '${ROOT_PACKAGE}.VERSION=${VERSION}' -X '${ROOT_PACKAGE}.REVISION=${REVISION}'
GO_LDFLAGS_VERSION := -X 'main.VERSION=${VERSION}' -X 'main.REVISION=${REVISION}'
GO_LDFLAGS := $(GO_LDFLAGS_VERSION)
GO_BUILD := -ldflags "$(GO_LDFLAGS)"

.PHONY: all
all: build run

.PHONY: build
build: interceptor.go
	@go build $(GO_BUILD) -o $(BINDIR)/interceptor -v

.PHONY: run
run: build
	@sudo ./bin/interceptor

.PHONY: clean
clean:
	@go clean
	@rm -rf $(BINDIR)
