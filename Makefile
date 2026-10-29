.PHONY: build test testacc fmt install help

# Latest release tag (without the "v"); override with `make install VERSION=x.y.z`
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//')
# Terraform's platform (e.g. darwin_arm64), which may differ from Go's default target
OS_ARCH := $(shell terraform version -json 2>/dev/null | sed -n 's/.*"platform": *"\([^"]*\)".*/\1/p')
OS_ARCH := $(if $(OS_ARCH),$(OS_ARCH),$(shell go env GOOS)_$(shell go env GOARCH))
PLUGIN_DIR := ~/.terraform.d/plugins/registry.terraform.io/arize-ai/arize/$(VERSION)/$(OS_ARCH)

build:
	go build -ldflags "-X main.version=$(VERSION)" -o bin/terraform-provider-arize .

test:
	go test -v ./...

testacc:
	TF_ACC=1 go test -v ./...

fmt:
	gofmt -s -w .

install:
	mkdir -p $(PLUGIN_DIR)
	GOOS=$(word 1,$(subst _, ,$(OS_ARCH))) GOARCH=$(word 2,$(subst _, ,$(OS_ARCH))) \
		go build -ldflags "-X main.version=$(VERSION)" -o $(PLUGIN_DIR)/terraform-provider-arize .

help:
	@echo "Usage: make [target]"
	@echo "Targets:"
	@echo "  build    - Build the provider"
	@echo "  test     - Run unit tests"
	@echo "  testacc  - Run acceptance tests (requires TF_ACC=1)"
	@echo "  fmt      - Format code"
	@echo "  install  - Build and install provider locally (VERSION=$(VERSION), $(OS_ARCH))"
