PACKAGES = $(shell go list ./... | grep -v '/vendor/')

all: bytomd bytomcli test

bytomd:
	@echo "Building bytomd to cmd/bytomd/bytomd"
	@go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/bytomd/bytomd cmd/bytomd/main.go

bytomcli:
	@echo "Building bytomcli to cmd/bytomcli/bytomcli"
	@go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/bytomcli/bytomcli cmd/bytomcli/main.go

test:
	@echo "====> Running go test"
	@go test $(PACKAGES)

.PHONY: test
