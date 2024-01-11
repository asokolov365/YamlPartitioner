MAKEFLAGS += --warn-undefined-variables
SHELL := /bin/bash
.SHELLFLAGS := -o pipefail -euc

GOLANGCI_LINT_VERSION='v1.51.2'
YP_VERSION?=$(shell cat version/VERSION)

GO_MODULES := $(shell find . -name go.mod -exec dirname {} \; | sort)

GOTEST_FLAGS := -v -bench

GOTAGS ?=
GOPATH := $(shell go env GOPATH)
GOARCH ?= $(shell go env GOARCH)
MAIN_GOPATH := $(shell go env GOPATH | cut -d: -f1)

export PATH := $(PWD)/bin:$(GOPATH)/bin:$(PATH)

# Get the git commit
GIT_COMMIT ?= $(shell git rev-parse --short HEAD)
GIT_DIRTY ?= $(shell test -n "`git status --porcelain`" && echo "+CHANGES" || true)
GIT_IMPORT := github.com/asokolov365/YamlPartitioner/version
GOLDFLAGS := -X $(GIT_IMPORT).GitCommit=$(GIT_COMMIT)$(GIT_DIRTY) -X $(GIT_IMPORT).Version=$(YP_VERSION)

ifeq (, $(shell which golangci-lint))
$(warning "unable to find golangci-lint in $(PATH), run: make tools")
endif

ifeq (, $(shell which gci))
$(warning "unable to find gci in $(PATH), run: make tools")
endif

export GIT_COMMIT
export GIT_DIRTY
export GOTAGS
export GOLDFLAGS

default: all

##@ Build

.PHONY: all
all: dev-build ## Command running by default

# used to make integration dependencies conditional
noop: ;

.PHONY: dev
dev: dev-build ## Dev creates binaries for testing locally - these are put into ./bin

.PHONY: dev-build
dev-build: ## Same as dev
	mkdir -p bin
	CGO_ENABLED=0 go install -ldflags "$(GOLDFLAGS)" -tags "$(GOTAGS)"
	# rm needed due to signature caching (https://apple.stackexchange.com/a/428388)
	rm -f ./bin/yp
	cp $(MAIN_GOPATH)/bin/YamlPartitioner ./bin/yp

.PHONY: go-mod-tidy
go-mod-tidy: $(foreach mod,$(GO_MODULES),go-mod-tidy/$(mod)) ## Run go mod tidy in every module

.PHONY: mod-tidy/%
go-mod-tidy/%:
	@echo "--> Running go mod tidy ($*)"
	@cd $* && go mod tidy

linux:  ## Linux builds a linux binary compatible with the source platform
	@mkdir -p ./pkg/bin/linux_$(GOARCH)
	CGO_ENABLED=0 GOOS=linux GOARCH=$(GOARCH) go build -o ./pkg/bin/linux_$(GOARCH) -ldflags "$(GOLDFLAGS)" -tags "$(GOTAGS)"



##@ Checks

.PHONY: fmt
fmt: $(foreach mod,$(GO_MODULES),fmt/$(mod)) ## Format go modules

.PHONY: fmt/%
fmt/%:
	@echo "--> Running go fmt ($*)"
	@cd $* && gofmt -s -l -w . && gofumpt -l -w .

.PHONY: fieldalignment
fieldalignment: $(foreach mod,$(GO_MODULES),fieldalignment/$(mod)) ## fieldalignment go modules

.PHONY: fieldalignment/%
fieldalignment/%:
	@echo "--> Running fieldalignment -fix ($*)"
	@cd $* && fieldalignment -fix ./...

.PHONY: lint
lint: $(foreach mod,$(GO_MODULES),lint/$(mod)) ## Lint go modules and test deps

.PHONY: lint/%
lint/%:
	@echo "--> Running golangci-lint ($*)"
	@cd $* && GOWORK=off CGO_ENABLED=0 golangci-lint run --build-tags '$(GOTAGS)' ./...
	@echo "--> Running enumcover ($*)"
	@cd $* && GOWORK=off enumcover ./...


##@ Testing

.PHONY: cover
cover: dev-build ## Run tests and generate coverage report
	go test -tags '$(GOTAGS)' ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

.PHONY: test
test: dev-build lint test-all

.PHONY: test-all
test-all: lint $(foreach mod,$(GO_MODULES),test-module/$(mod)) ## Test all

.PHONY: test-module/%
test-module/%:
	@echo "--> Running go test ($*)"
	cd $* && go test $(GOTEST_FLAGS) -tags '$(GOTAGS)' ./...

.PHONY: test-race
test-race: ## Test race
	$(MAKE) test GOTEST_FLAGS="-v -race"


##@ Dependencies

.PHONY: deps
deps: ## Installs Go dependencies.
	@echo "--> Running go get -v"
	go get -v ./...


##@ Tools

.PHONY: tools
tools: gci golangci-lint gofumpt ## Installs various supporting Go tools.

.PHONY: gci
gci:
	@echo "--> Installing github.com/daixiang0/gci@latest"
	go install github.com/daixiang0/gci@latest

.PHONY: golangci-lint
golangci-lint:
	@echo "--> Installing golangci-lint $(GOLANGCI_LINT_VERSION)"
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin $(GOLANGCI_LINT_VERSION)

.PHONY: gofumpt
gofumpt:
	@echo "--> Installing mvdan.cc/gofumpt@latest"
	go install mvdan.cc/gofumpt@latest

print-%  : ; @echo $($*) ## utility to echo a makefile variable (i.e. 'make print-GOPATH')

.PHONY: module-versions
module-versions: ## Print a list of modules which can be updated. Columns are: module current_version date_of_current_version latest_version
	@go list -m -u -f '{{if .Update}} {{printf "%-50v %-40s" .Path .Version}} {{with .Time}} {{ .Format "2006-01-02" -}} {{else}} {{printf "%9s" ""}} {{end}}   {{ .Update.Version}} {{end}}' all


##@ Cleanup

.PHONY: clean
clean: ## Removes produced binaries, libraries, and temp files
	@rm -rf build release cover vendor
	@rm -f test.log exit-code coverage.out
	@docker rmi -f yp_build > /dev/null 2>&1 || true
	./scripts/test.sh clean

##@ Help

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php
.PHONY: help
help: ## Display this help.
	@echo -e "\033[32m"
	@echo "Targets in this Makefile build and test YamlPartitioner in a build container in"
	@echo "Docker. For testing (only), use the 'local' prefix target to run targets directly"
	@echo "on your workstation (ex. 'make local test'). You will need to have its GOPATH set"
	@echo "and have already run 'make tools'. Set GOOS=linux to build binaries for Docker."
	@echo "Do not use 'make local' for building binaries for public release!"
	@echo "Before packaging always run 'make clean build test integration'!"
	@echo
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
