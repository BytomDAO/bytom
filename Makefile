GOTOOLS  = \
		   github.com/mitchellh/gox \
		   github.com/Masterminds/glide
PACKAGES = $(shell go list ./... | grep -v '/vendor/' | grep -v '/rpc/')

all: install test

bytomd:
	@echo "Building bytomd to cmd/bytomd/bytomd"
	@go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/bytomd/bytomd cmd/bytomd/main.go

bytomcli:
	@echo "Building bytomcli to cmd/bytomcli/bytomcli"
	@go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/bytomcli/bytomcli cmd/bytomcli/main.go

install: get_vendor_deps
	@echo "====> Done!"

get_vendor_deps: ensure_tools
	@rm -rf vendor/
	@echo "====> Running glide install"
	@glide install

ensure_tools:
	go get $(GOTOOLS)

test:
	@echo "====> Running go test"
	@go test $(PACKAGES)

.PHONY: install get_vendor_deps ensure_tools test
