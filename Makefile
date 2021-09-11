LOCAL_BINDIR := bin
SYSTEM_BINDIR := /usr/bin
SYSTEM_LIBDIR := /var/lib/telescreen
LIBPCAP := /path/to/libpcap
VERSION := $(shell git describe --tags --abbrev=0)
REVISION := $(shell git rev-parse --short HEAD)
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
GO_TAGS = -tags netgo -installsuffix netgo
GO_LDFLAGS_VERSION := -X 'main.VERSION=$(VERSION)' -X 'main.REVISION=$(REVISION)'
GO_LDFLAGS_STATICLINK := -linkmode external -extldflags -static -L $(LIBPCAP)
GO_LDFLAGS := -s -w $(GO_LDFLAGS_VERSION)
GO_BUILD_DYNAMIC := $(GO_TAGS) -ldflags "$(GO_LDFLAGS)" -v
GO_BUILD_STATIC := $(GO_TAGS) -ldflags "$(GO_LDFLAGS) $(GO_LDFLAGS_STATICLINK)" -v
GO_SRC := cmd/telescreen/main.go
GO_BIN := telescreen
GO_BIN_STATIC := telescreen_$(GOOS)_$(GOARCH)_$(VERSION)-$(REVISION)
DOCKER_IMAGE_TAG := wide-vsix/telescreen:$(VERSION)-$(REVISION)
DEFAULT_TELESCREEN_DEVICE := vsix

.PHONY: all
all: build

.PHONY: build
build:
	@mkdir -p $(LOCAL_BINDIR)
	@go build $(GO_BUILD_DYNAMIC) -o $(LOCAL_BINDIR)/$(GO_BIN) $(GO_SRC)

.PHONY: build-static
build-static: $(LIBPCAP)
	@mkdir -p $(LOCAL_BINDIR)
	@go build $(GO_BUILD_STATIC) -o $(LOCAL_BINDIR)/$(GO_BIN_STATIC) $(GO_SRC)
	@cp $(LOCAL_BINDIR)/$(GO_BIN_STATIC) $(LOCAL_BINDIR)/$(GO_BIN)

.PHONY: build-static-docker
build-static-docker:
	@mkdir -p $(LOCAL_BINDIR)
	@DOCKER_BUILDKIT=0 docker build \
		--build-arg TELESCREEN_VERSION=$(VERSION) --build-arg TELESCREEN_REVISION=$(REVISION) \
		--tag $(DOCKER_IMAGE_TAG) .
	@docker rm -f telescreen-binary-copy 2>/dev/null || true
	@docker create -it --name telescreen-binary-copy $(DOCKER_IMAGE_TAG)
	@docker cp telescreen-binary-copy:/work/telescreen $(LOCAL_BINDIR)/$(GO_BIN_STATIC)
	@docker rm -f telescreen-binary-copy
	@cp $(LOCAL_BINDIR)/$(GO_BIN_STATIC) $(LOCAL_BINDIR)/$(GO_BIN)

.PHONY: clean
clean:
	@go clean
	@rm -rf $(LOCAL_BINDIR)

.PHONY: install
install: $(LOCAL_BINDIR)/$(GO_BIN)
	@sudo cp $(LOCAL_BINDIR)/$(GO_BIN) $(SYSTEM_BINDIR)/$(GO_BIN)

.PHONY: install-agent
install-agent: $(SYSTEM_BINDIR)/$(GO_BIN) .secrets/db_password.txt
	@sudo mkdir -p $(SYSTEM_LIBDIR)
	@sudo cp docker-compose.yml $(SYSTEM_LIBDIR)/docker-compose.yml
	@sudo cp -r .secrets $(SYSTEM_LIBDIR)/.secrets
	@sudo cp systemd/telescreen@.service /etc/systemd/system/telescreen@.service
	@sudo systemctl daemon-reload
	@sudo systemctl enable --now telescreen@$(DEFAULT_TELESCREEN_DEVICE).service

.PHONY: install-database
install-database: .secrets/db_password.txt
	@sudo mkdir -p $(SYSTEM_LIBDIR)
	@sudo cp docker-compose.yml $(SYSTEM_LIBDIR)/docker-compose.yml
	@sudo cp -r .secrets $(SYSTEM_LIBDIR)/.secrets
	@sudo docker-compose -f $(SYSTEM_LIBDIR)/docker-compose.yml up -d postgres

.PHONY: uninstall
uninstall:
	@sudo systemctl disable --now telescreen@$(DEFAULT_TELESCREEN_DEVICE).service || true
	@sudo docker-compose -f $(SYSTEM_LIBDIR)/docker-compose.yml kill || true
	@sudo docker-compose -f $(SYSTEM_LIBDIR)/docker-compose.yml rm -f || true
	@sudo docker volume rm telescreen_psql telescreen_pgadmin || true
	@sudo rm -rf $(SYSTEM_BINDIR)/telescreen $(SYSTEM_LIBDIR) || true
	@sudo rm -f /etc/systemd/system/telescreen@.service || true
	@sudo systemctl daemon-reload
