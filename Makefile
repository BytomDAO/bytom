GOTOOLS  = \
		   github.com/mitchellh/gox \
		   github.com/Masterminds/glide
PACKAGES = $(shell go list ./... | grep -v '/vendor/' | grep -v '/rpc/')

all: install test

install: get_vendor_deps
	@go install --ldflags '-extldflags "-static"' \
		--ldflags "-X github.com/Bytom/blockchain/version.GitCommit=`git rev-parse HEAD`" ./node/
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
