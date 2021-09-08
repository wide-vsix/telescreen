BINDIR := bin
VERSION := $(shell git describe --tags --abbrev=0)
REVISION := $(shell git rev-parse --short HEAD)
GOVERSION := $(shell go version)
LIBPCAP := /path/to/libpcap
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
GO_TAGS = -tags netgo -installsuffix netgo
GO_LDFLAGS_VERSION := -X 'main.VERSION=$(VERSION)' -X 'main.REVISION=$(REVISION)'
GO_LDFLAGS_STATICLINK := -linkmode external -extldflags -static -L $(LIBPCAP)
GO_LDFLAGS := -s -w $(GO_LDFLAGS_VERSION)
GO_BUILD_DYNAMIC := $(GO_TAGS) -ldflags "$(GO_LDFLAGS)" -v
GO_BUILD_STATIC := $(GO_TAGS) -ldflags "$(GO_LDFLAGS) $(GO_LDFLAGS_STATICLINK)" -v
GO_BINARYNAME_STATIC := telescreen_$(GOOS)_$(GOARCH)_$(VERSION)-$(REVISION)
DOCKER_IMAGE_TAG := wide-vsix/telescreen:$(VERSION)-$(REVISION)

.PHONY: all
all: build

.PHONY: build
build:
	@go build $(GO_BUILD_DYNAMIC) -o $(BINDIR)/telescreen ./cmd/telescreen/main.go

.PHONY: static-build $(LIBPCAP)
static-build:
	@go build $(GO_BUILD_STATIC) -o $(BINDIR)/$(GO_BINARYNAME_STATIC) ./cmd/telescreen/main.go

.PHONY: docker-build
docker-build:
	@DOCKER_BUILDKIT=0 docker build \
		--build-arg telescreen_VERSION=$(VERSION) --build-arg telescreen_REVISION=$(REVISION) \
		--tag $(DOCKER_IMAGE_TAG) .
	@docker rm -f telescreen-binary-copy 2>/dev/null || true
	@docker create -it --name telescreen-binary-copy $(DOCKER_IMAGE_TAG)
	@docker cp telescreen-binary-copy:/work/telescreen $(BINDIR)/$(GO_BINARYNAME_STATIC)
	@docker rm -f telescreen-binary-copy

.PHONY: clean
clean:
	@go clean
	@rm -rf $(BINDIR)

.PHONY: install
install:
	@sudo cp bin/telescreen /usr/local/bin/telescreen
	@sudo mkdir -p /var/lib/telescreen
	@sudo cp docker-compose.yml /var/lib/telescreen/docker-compose.yml
	@sudo cp -r .secrets /var/lib/telescreen/.secrets
	@sudo cp systemd/telescreen@.service /etc/systemd/system/telescreen@.service
	@sudo systemctl daemon-reload
	@sudo systemctl enable telescreen@vsix.service

.PHONY: install-db
install-db:
	@sudo cp bin/telescreen /usr/local/bin/telescreen
	@sudo mkdir -p /var/lib/telescreen
	@sudo cp docker-compose.yml /var/lib/telescreen/docker-compose.yml
	@sudo cp -r .secrets /var/lib/telescreen/.secrets
	@sudo docker-compose -f /var/lib/telescreen/docker-compose.yml up -d postgres

.PHONY: uninstall
uninstall:
	@sudo systemctl disable --now telescreen@vsix.service || true
	@sudo docker-compose -f /var/lib/telescreen/docker-compose.yml kill || true
	@sudo docker-compose -f /var/lib/telescreen/docker-compose.yml rm -f || true
	@sudo docker volume rm telescreen_psql telescreen_pgadmin || true
	@sudo rm -rf /usr/local/bin/telescreen /var/lib/telescreen || true
	@sudo rm -f /etc/systemd/system/telescreen@.service || true
	@sudo systemctl daemon-reload
