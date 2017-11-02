GOTOOLS  = github.com/mitchellh/gox
PACKAGES = $(shell go list ./... | grep -v '/vendor/' | grep -v '/rpc/')

all: test

bytomd:
	@echo "Building bytomd to cmd/bytomd/bytomd"
	@go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/bytomd/bytomd cmd/bytomd/main.go

bytomcli:
	@echo "Building bytomcli to cmd/bytomcli/bytomcli"
	@go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/bytomcli/bytomcli cmd/bytomcli/main.go

ensure_tools:
	go get $(GOTOOLS)

test:
	@echo "====> Running go test"
	@go test $(PACKAGES)

.PHONY: ensure_tools test
