ifndef GOOS
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	GOOS := darwin
	CC = /usr/local/opt/llvm/bin/clang++
else ifeq ($(UNAME_S),Linux)
	GOOS := linux
	CC = g++
else
$(error "$$GOOS is not defined. If you are using Windows, try to re-make using 'GOOS=windows make ...' ")
endif
endif

PACKAGES    := $(shell go list ./... | grep -v '/vendor/' | grep -v '/crypto/ed25519/chainkd' | grep -v 'github.com/bytom/mining/tensority/legacy' | grep -v 'github.com/bytom/mining/tensority/stlib' | grep -v 'github.com/bytom/mining/tensority/dylib')

BUILD_FLAGS := -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`"

MINER_BINARY32 := miner-$(GOOS)_386
MINER_BINARY64 := miner-$(GOOS)_amd64

BYTOMD_BINARY32 := bytomd-$(GOOS)_386
BYTOMD_BINARY64 := bytomd-$(GOOS)_amd64

BYTOMCLI_BINARY32 := bytomcli-$(GOOS)_386
BYTOMCLI_BINARY64 := bytomcli-$(GOOS)_amd64

VERSION := $(shell awk -F= '/Version =/ {print $$2}' version/version.go | tr -d "\" ")

MINER_RELEASE32 := miner-$(VERSION)-$(GOOS)_386
MINER_RELEASE64 := miner-$(VERSION)-$(GOOS)_amd64

BYTOMD_RELEASE32 := bytomd-$(VERSION)-$(GOOS)_386
BYTOMD_RELEASE64 := bytomd-$(VERSION)-$(GOOS)_amd64

BYTOMCLI_RELEASE32 := bytomcli-$(VERSION)-$(GOOS)_386
BYTOMCLI_RELEASE64 := bytomcli-$(VERSION)-$(GOOS)_amd64

BYTOM_RELEASE32 := bytom-$(VERSION)-$(GOOS)_386
BYTOM_RELEASE64 := bytom-$(VERSION)-$(GOOS)_amd64

all: test target release-all

bytomd-simd:
	@echo "Building bytomd to cmd/bytomd/bytomd for $(GOOS)"
	rm -f mining/tensority/*.go
	cp mining/tensority/stlib/*.go mining/tensority/
	$(CC) -o mining/tensority/stlib/cSimdTs.o -c mining/tensority/stlib/cSimdTs.cpp -std=c++11 -pthread -mavx2 -O3 -fopenmp -D_USE_OPENMP
	go build $(BUILD_FLAGS) -o cmd/bytomd/bytomd cmd/bytomd/main.go
	rm -f mining/tensority/*.go
	cp mining/tensority/legacy/*.go mining/tensority/


bytomd:
	@echo "Building bytomd to cmd/bytomd/bytomd"
	@rm -f mining/tensority/*.go
	@cp mining/tensority/legacy/*.go mining/tensority/
	@go build $(BUILD_FLAGS) -o cmd/bytomd/bytomd cmd/bytomd/main.go

bytomcli:
	@echo "Building bytomcli to cmd/bytomcli/bytomcli"
	@rm -f mining/tensority/*.go
	@cp mining/tensority/legacy/*.go mining/tensority/
	@go build $(BUILD_FLAGS) -o cmd/bytomcli/bytomcli cmd/bytomcli/main.go

target:
	mkdir -p $@

binary: target/$(BYTOMD_BINARY32) target/$(BYTOMD_BINARY64) target/$(BYTOMCLI_BINARY32) target/$(BYTOMCLI_BINARY64) target/$(MINER_BINARY32) target/$(MINER_BINARY64)

ifeq ($(GOOS),windows)
release: binary
	cd target && cp -f $(MINER_BINARY32) $(MINER_BINARY32).exe
	cd target && cp -f $(BYTOMD_BINARY32) $(BYTOMD_BINARY32).exe
	cd target && cp -f $(BYTOMCLI_BINARY32) $(BYTOMCLI_BINARY32).exe
	cd target && md5sum $(MINER_BINARY32).exe $(BYTOMD_BINARY32).exe $(BYTOMCLI_BINARY32).exe >$(BYTOM_RELEASE32).md5
	cd target && zip $(BYTOM_RELEASE32).zip $(MINER_BINARY32).exe $(BYTOMD_BINARY32).exe $(BYTOMCLI_BINARY32).exe $(BYTOM_RELEASE32).md5
	cd target && rm -f $(MINER_BINARY32) $(BYTOMD_BINARY32) $(BYTOMCLI_BINARY32) $(MINER_BINARY32).exe $(BYTOMD_BINARY32).exe $(BYTOMCLI_BINARY32).exe $(BYTOM_RELEASE32).md5
	cd target && cp -f $(MINER_BINARY64) $(MINER_BINARY64).exe
	cd target && cp -f $(BYTOMD_BINARY64) $(BYTOMD_BINARY64).exe
	cd target && cp -f $(BYTOMCLI_BINARY64) $(BYTOMCLI_BINARY64).exe
	cd target && md5sum $(MINER_BINARY64).exe $(BYTOMD_BINARY64).exe $(BYTOMCLI_BINARY64).exe >$(BYTOM_RELEASE64).md5
	cd target && zip $(BYTOM_RELEASE64).zip $(MINER_BINARY64).exe $(BYTOMD_BINARY64).exe $(BYTOMCLI_BINARY64).exe $(BYTOM_RELEASE64).md5
	cd target && rm -f $(MINER_BINARY64) $(BYTOMD_BINARY64) $(BYTOMCLI_BINARY64) $(MINER_BINARY64).exe $(BYTOMD_BINARY64).exe $(BYTOMCLI_BINARY64).exe $(BYTOM_RELEASE64).md5
else
release: binary
	cd target && md5sum $(MINER_BINARY32) $(BYTOMD_BINARY32) $(BYTOMCLI_BINARY32) >$(BYTOM_RELEASE32).md5
	cd target && tar -czf $(BYTOM_RELEASE32).tgz $(MINER_BINARY32) $(BYTOMD_BINARY32) $(BYTOMCLI_BINARY32) $(BYTOM_RELEASE32).md5
	cd target && rm -f $(MINER_BINARY32) $(BYTOMD_BINARY32) $(BYTOMCLI_BINARY32) $(BYTOM_RELEASE32).md5
	cd target && md5sum $(MINER_BINARY64) $(BYTOMD_BINARY64) $(BYTOMCLI_BINARY64) >$(BYTOM_RELEASE64).md5
	cd target && tar -czf $(BYTOM_RELEASE64).tgz $(MINER_BINARY64) $(BYTOMD_BINARY64) $(BYTOMCLI_BINARY64) $(BYTOM_RELEASE64).md5
	cd target && rm -f $(MINER_BINARY64) $(BYTOMD_BINARY64) $(BYTOMCLI_BINARY64) $(BYTOM_RELEASE64).md5
endif

release-all: clean
	GOOS=darwin  make release
	GOOS=linux   make release
	GOOS=windows make release

clean:
	@echo "Cleaning binaries built"
	@rm -rf target
	@rm -rf cmd/bytomcli/bytomcli
	@rm -rf cmd/miner/miner
	@rm -rf cmd/miner/*.dylib
	@rm -rf cmd/bytomd/bytomd
	@rm -rf cmd/bytomd/*.dylib
	@rm -rf mining/tensority/*.dylib
	@rm -rf mining/tensority/dylib/*.dylib
	@rm -rf mining/tensority/stlib/*.o
	@rm -f mining/tensority/*.go
	@cp mining/tensority/legacy/*.go mining/tensority/

target/$(BYTOMD_BINARY32):
	CGO_ENABLED=0 GOARCH=386 go build $(BUILD_FLAGS) -o $@ cmd/bytomd/main.go

target/$(BYTOMD_BINARY64):
	CGO_ENABLED=0 GOARCH=amd64 go build $(BUILD_FLAGS) -o $@ cmd/bytomd/main.go

target/$(BYTOMCLI_BINARY32):
	CGO_ENABLED=0 GOARCH=386 go build $(BUILD_FLAGS) -o $@ cmd/bytomcli/main.go

target/$(BYTOMCLI_BINARY64):
	CGO_ENABLED=0 GOARCH=amd64 go build $(BUILD_FLAGS) -o $@ cmd/bytomcli/main.go

target/$(MINER_BINARY32):
	CGO_ENABLED=0 GOARCH=386 go build $(BUILD_FLAGS) -o $@ cmd/miner/main.go

target/$(MINER_BINARY64):
	CGO_ENABLED=0 GOARCH=amd64 go build $(BUILD_FLAGS) -o $@ cmd/miner/main.go

test:
	@echo "====> Running go test"
	@rm -f mining/tensority/*.go
	@cp mining/tensority/legacy/*.go mining/tensority/
	@go test -tags "network" $(PACKAGES)

benchmark:
	@rm -f mining/tensority/*.go
	@cp mining/tensority/legacy/*.go mining/tensority/
	@go test -bench $(PACKAGES)

functional-tests:
	@rm -f mining/tensority/*.go
	@cp mining/tensority/legacy/*.go mining/tensority/
	@go test -v -timeout=5m -tags=functional ./test

ci: test functional-tests

.PHONY: all target release-all clean test benchmark




