BINDIR := bin
VERSION := $(shell git describe --tags --abbrev=0)
REVISION := $(shell git rev-parse --short HEAD)
GOVERSION := $(shell go version)
LIBPCAP := /path/to/libpcap
GO_TAGS = -tags netgo -installsuffix netgo
GO_LDFLAGS_VERSION := -X 'main.VERSION=${VERSION}' -X 'main.REVISION=${REVISION}'
GO_LDFLAGS_STATICLINK := -linkmode external -extldflags -static -L ${LIBPCAP}
GO_LDFLAGS := -s -w $(GO_LDFLAGS_VERSION)
GO_BUILD_DYNAMIC := $(GO_TAGS) -ldflags "$(GO_LDFLAGS)" -v
GO_BUILD_STATIC := $(GO_TAGS) -ldflags "$(GO_LDFLAGS) $(GO_LDFLAGS_STATICLINK)" -v
export GOOS := $(shell go env GOOS)
export GOARCH := $(shell go env GOARCH)

.PHONY: all
all: build

.PHONY: build
build:
	@go build $(GO_BUILD_DYNAMIC) -o $(BINDIR)/interceptor ./cmd/interceptor/main.go

.PHONY: static-build ${LIBPCAP}
static-build:
	@go build $(GO_BUILD_STATIC) -o $(BINDIR)/interceptor ./cmd/interceptor/main.go

.PHONY: docker-build
docker-build:
	@DOCKER_BUILDKIT=0 docker build \
		--build-arg INTERCEPTOR_VERSION=${VERSION} --build-arg INTERCEPTOR_REVISION=${REVISION} \
		--tag wide-vsix/dns-query-interceptor:${VERSION}-${REVISION} .

.PHONY: clean
clean:
	@go clean
	@rm -rf $(BINDIR)

.PHONY: install
install:
	@sudo cp bin/interceptor /usr/local/bin/interceptor
	@sudo mkdir -p /var/lib/dns-query-interceptor
	@sudo cp docker-compose.yml /var/lib/dns-query-interceptor/docker-compose.yml
	@sudo cp -r .secrets /var/lib/dns-query-inerceptor/.secrets
	@sudo cp systemd/dns-query-interceptor@.service /etc/systemd/system/dns-query-interceptor@.service
	@sudo systemctl daemon-reload
	@sudo systemctl enable dns-query-interceptor@vsix.service

.PHONY: install-db
install-db:
	@sudo cp bin/interceptor /usr/local/bin/interceptor
	@sudo mkdir -p /var/lib/dns-query-interceptor
	@sudo cp docker-compose.yml /var/lib/dns-query-interceptor/docker-compose.yml
	@sudo cp -r .secrets /var/lib/dns-query-inerceptor/.secrets
	@sudo docker-compose -f /var/lib/dns-query-interceptor/docker-compose.yml up -d postgres

.PHONY: uninstall
uninstall:
	@sudo systemctl disable --now dns-query-interceptor@vsix.service || true
	@sudo docker-compose -f /var/lib/dns-query-interceptor/docker-compose.yml kill || true
	@sudo docker-compose -f /var/lib/dns-query-interceptor/docker-compose.yml rm -f || true
	@sudo docker volume rm dns-query-interceptor_psql dns-query-interceptor_pgadmin || true
	@sudo rm -rf /usr/local/bin/interceptor /var/lib/dns-query-interceptor || true
	@sudo rm -f /etc/systemd/system/dns-query-interceptor@.service || true
	@sudo systemctl daemon-reload
