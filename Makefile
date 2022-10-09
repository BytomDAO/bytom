ifndef GOOS
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	GOOS := darwin
else ifeq ($(UNAME_S),Linux)
	GOOS := linux
else
	GOOS := windows
endif
endif

PACKAGES    := $(shell go list ./... | grep -v '/lib/')

BUILD_FLAGS := -ldflags "-X github.com/bytom/bytom/version.GitCommit=`git rev-parse HEAD`"

BYTOMD_BINARY32 := bytomd-$(GOOS)_386
BYTOMD_BINARY64 := bytomd-$(GOOS)_amd64

BYTOMCLI_BINARY32 := bytomcli-$(GOOS)_386
BYTOMCLI_BINARY64 := bytomcli-$(GOOS)_amd64

VERSION := $(shell awk -F= '/Version =/ {print $$2}' version/version.go | tr -d "\" ")

BYTOMD_RELEASE32 := bytomd-$(VERSION)-$(GOOS)_386
BYTOMD_RELEASE64 := bytomd-$(VERSION)-$(GOOS)_amd64

BYTOMCLI_RELEASE32 := bytomcli-$(VERSION)-$(GOOS)_386
BYTOMCLI_RELEASE64 := bytomcli-$(VERSION)-$(GOOS)_amd64

BYTOM_RELEASE32 := bytom-$(VERSION)-$(GOOS)_386
BYTOM_RELEASE64 := bytom-$(VERSION)-$(GOOS)_amd64

all: test target release-all install

bytomd:
	@echo "Building bytomd to cmd/bytomd/bytomd"
	@go build $(BUILD_FLAGS) -o cmd/bytomd/bytomd cmd/bytomd/main.go

bytomcli:
	@echo "Building bytomcli to cmd/bytomcli/bytomcli"
	@go build $(BUILD_FLAGS) -o cmd/bytomcli/bytomcli cmd/bytomcli/main.go

install:
	@echo "Installing bytomd and bytomcli to $(GOPATH)/bin"
	@go install ./cmd/bytomd
	@go install ./cmd/bytomcli

target:
	mkdir -p $@

binary: target/$(BYTOMD_BINARY32) target/$(BYTOMD_BINARY64) target/$(BYTOMCLI_BINARY32) target/$(BYTOMCLI_BINARY64)

ifeq ($(GOOS),windows)
release: binary
	cd target && cp -f $(BYTOMD_BINARY32) $(BYTOMD_BINARY32).exe
	cd target && cp -f $(BYTOMCLI_BINARY32) $(BYTOMCLI_BINARY32).exe
	cd target && md5sum  $(BYTOMD_BINARY32).exe $(BYTOMCLI_BINARY32).exe >$(BYTOM_RELEASE32).md5
	cd target && zip $(BYTOM_RELEASE32).zip  $(BYTOMD_BINARY32).exe $(BYTOMCLI_BINARY32).exe $(BYTOM_RELEASE32).md5
	cd target && rm -f  $(BYTOMD_BINARY32) $(BYTOMCLI_BINARY32)  $(BYTOMD_BINARY32).exe $(BYTOMCLI_BINARY32).exe $(BYTOM_RELEASE32).md5
	cd target && cp -f $(BYTOMD_BINARY64) $(BYTOMD_BINARY64).exe
	cd target && cp -f $(BYTOMCLI_BINARY64) $(BYTOMCLI_BINARY64).exe
	cd target && md5sum  $(BYTOMD_BINARY64).exe $(BYTOMCLI_BINARY64).exe >$(BYTOM_RELEASE64).md5
	cd target && zip $(BYTOM_RELEASE64).zip  $(BYTOMD_BINARY64).exe $(BYTOMCLI_BINARY64).exe $(BYTOM_RELEASE64).md5
	cd target && rm -f  $(BYTOMD_BINARY64) $(BYTOMCLI_BINARY64)  $(BYTOMD_BINARY64).exe $(BYTOMCLI_BINARY64).exe $(BYTOM_RELEASE64).md5
else
release: binary
	cd target && md5sum  $(BYTOMD_BINARY32) $(BYTOMCLI_BINARY32) >$(BYTOM_RELEASE32).md5
	cd target && tar -czf $(BYTOM_RELEASE32).tgz  $(BYTOMD_BINARY32) $(BYTOMCLI_BINARY32) $(BYTOM_RELEASE32).md5
	cd target && rm -f  $(BYTOMD_BINARY32) $(BYTOMCLI_BINARY32) $(BYTOM_RELEASE32).md5
	cd target && md5sum  $(BYTOMD_BINARY64) $(BYTOMCLI_BINARY64) >$(BYTOM_RELEASE64).md5
	cd target && tar -czf $(BYTOM_RELEASE64).tgz  $(BYTOMD_BINARY64) $(BYTOMCLI_BINARY64) $(BYTOM_RELEASE64).md5
	cd target && rm -f  $(BYTOMD_BINARY64) $(BYTOMCLI_BINARY64) $(BYTOM_RELEASE64).md5
endif

release-all: clean
	GOOS=darwin  make release
	GOOS=linux   make release
	GOOS=windows make release

clean:
	@echo "Cleaning binaries built..."
	@rm -rf cmd/bytomd/bytomd
	@rm -rf cmd/bytomcli/bytomcli
	@rm -rf target
	@rm -rf $(GOPATH)/bin/bytomd
	@rm -rf $(GOPATH)/bin/bytomcli
	@echo "Cleaning temp test data..."
	@rm -rf test/pseudo_hsm*
	@rm -rf blockchain/pseudohsm/testdata/pseudo/
	@echo "Cleaning sm2 pem files..."
	@rm -rf crypto/sm2/*.pem
	@echo "Done."

target/$(BYTOMD_BINARY32):
	CGO_ENABLED=0 GOARCH=386 go build $(BUILD_FLAGS) -o $@ cmd/bytomd/main.go

target/$(BYTOMD_BINARY64):
	CGO_ENABLED=0 GOARCH=amd64 go build $(BUILD_FLAGS) -o $@ cmd/bytomd/main.go

target/$(BYTOMCLI_BINARY32):
	CGO_ENABLED=0 GOARCH=386 go build $(BUILD_FLAGS) -o $@ cmd/bytomcli/main.go

target/$(BYTOMCLI_BINARY64):
	CGO_ENABLED=0 GOARCH=amd64 go build $(BUILD_FLAGS) -o $@ cmd/bytomcli/main.go

test:
	@echo "====> Running go test"
	@go test $(PACKAGES)

benchmark:
	@go test -bench $(PACKAGES)

functional-tests:
	@go test -timeout=5m -tags="functional" ./test 

ci: test

.PHONY: all target release-all clean test benchmark
