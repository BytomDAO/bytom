GOTOOLS  = github.com/mitchellh/gox \
		   github.com/Masterminds/glide
PACKAGES = $(shell go list ./... | grep -v '/vendor/')

all: install test

install: get_vendor_deps copy
	@go install --ldflags '-extldflags "-static"' \
		--ldflags "-X github.com/Bytom/blockchain/version.GitCommit=`git rev-parse HEAD`" ./node/
	@echo "====> Done!"

get_vendor_deps: ensure_tools
	@rm -rf vendor/
	@echo "====> Running glide install"
	@glide install

ensure_tools:
	go get $(GOTOOLS)

# In case of the terrible network condition
copy:
	@cp -r vendor/github.com/golang/crypto vendor/golang.org/x/crypto
	@cp -r vendor/github.com/golang/net    vendor/golang.org/x/net
	@cp -r vendor/github.com/golang/text   vendor/golang.org/x/text
	@cp -r vendor/github.com/golang/tools  vendor/golang.org/x/tools
	@cp -r vendor/github.com/golang/time   vendor/golang.org/x/time

test:
	@echo "=====> Running go test"
	@go test $(PACKAGES)

.PHONY: install get_vendor_deps ensure_tools copy test
