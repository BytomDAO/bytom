PACKAGES = $(shell go list ./... | grep -v '/vendor/')

all: bytomd bytomcli miner test

bytomd:
	@echo "Building bytomd to cmd/bytomd/bytomd"
	@go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/bytomd/bytomd cmd/bytomd/main.go

bytomcli:
	@echo "Building bytomcli to cmd/bytomcli/bytomcli"
	@go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/bytomcli/bytomcli cmd/bytomcli/main.go

miner:
	@echo "Building miner to cmd/miner/miner"
	@go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/miner/miner cmd/miner/main.go

multi_platform: bytomd bytomcli
	@echo "Building multi platform binary"
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/bytomcli/darwin/bytomcli cmd/bytomcli/main.go
	@echo "Building bytomd to cmd/bytomd/bytomd"
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/bytomd/darwin/bytomd cmd/bytomd/main.go

	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/bytomcli/windows/bytomcli cmd/bytomcli/main.go
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/bytomd/windows/bytomd cmd/bytomd/main.go

test:
	@echo "====> Running go test"
	@go test $(PACKAGES)

benchmark:
	go test -bench $(PACKAGES)

.PHONY: test
