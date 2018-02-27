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

multi_platform:
	@echo "Building multi platform binary"
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/darwin/bytomcli cmd/bytomcli/main.go
	@echo "Building bytomd to cmd/bytomd/bytomd"
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/darwin/bytomd cmd/bytomd/main.go
	@echo "Building miner to cmd/miner/miner"
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/darwin/miner cmd/miner/main.go

	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/windows/bytomcli cmd/bytomcli/main.go
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/windows/bytomd cmd/bytomd/main.go
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/windows/miner cmd/miner/main.go

	go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/ubuntu64/bytomcli cmd/bytomcli/main.go
	go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/ubuntu64/bytomd cmd/bytomd/main.go
	go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/ubuntu64/miner cmd/miner/main.go

386_multi_platform:
	@echo "Building multi platform binary"
	CGO_ENABLED=0 GOOS=darwin GOARCH=386 go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/darwin386/bytomcli cmd/bytomcli/main.go
	@echo "Building bytomd to cmd/bytomd/bytomd"
	CGO_ENABLED=0 GOOS=darwin GOARCH=386 go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/darwin386/bytomd cmd/bytomd/main.go
	@echo "Building miner to cmd/miner/miner"
	CGO_ENABLED=0 GOOS=darwin GOARCH=386 go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/darwin386/miner cmd/miner/main.go

	CGO_ENABLED=0 GOOS=windows GOARCH=386 go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/windows386/bytomcli cmd/bytomcli/main.go
	CGO_ENABLED=0 GOOS=windows GOARCH=386 go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/windows386/bytomd cmd/bytomd/main.go
	CGO_ENABLED=0 GOOS=windows GOARCH=386 go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/windows386/miner cmd/miner/main.go

	GOOS=linux GOARCH=386 go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/linux386/bytomcli cmd/bytomcli/main.go
	GOOS=linux GOARCH=386 go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/linux386/bytomd cmd/bytomd/main.go
	GOOS=linux GOARCH=386 go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" \
    -o cmd/linux386/miner cmd/miner/main.go

test:
	@echo "====> Running go test"
	@go test -tags "network" $(PACKAGES)

benchmark:
	go test -bench $(PACKAGES)

.PHONY: test
