.PHONY: build test testacc fmt install help

build:
	go build -o bin/terraform-provider-arize .

test:
	go test -v ./...

testacc:
	TF_ACC=1 go test -v ./...

fmt:
	gofmt -s -w .

install: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/arize-ai/arize/0.0.1/darwin_amd64
	cp bin/terraform-provider-arize ~/.terraform.d/plugins/registry.terraform.io/arize-ai/arize/0.0.1/darwin_amd64/

help:
	@echo "Usage: make [target]"
	@echo "Targets:"
	@echo "  build    - Build the provider"
	@echo "  test     - Run unit tests"
	@echo "  testacc  - Run acceptance tests (requires TF_ACC=1)"
	@echo "  fmt      - Format code"
	@echo "  install  - Build and install provider locally"
