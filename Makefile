PKG=github.com/manifoldco/heighliner
API_VERSIONS=$(sort $(patsubst pkg/api/%/,%,$(dir $(wildcard pkg/api/*/))))

ci: lint test
.PHONY: ci

#################################################
# Bootstrapping for base golang package deps
#################################################
BOOTSTRAP=\
	github.com/golang/dep/cmd/dep

$(BOOTSTRAP):
	go get -u $@

bootstrap: $(BOOTSTRAP)
	gometalinter --install

vendor: Gopkg.lock
	dep ensure -v -vendor-only

.PHONY: bootstrap $(BOOTSTRAP) vendor

#################################################
# Testing and linting
#################################################
LINTERS=\
	gofmt \
	golint \
	gosimple \
	vet \
	misspell \
	ineffassign \
	deadcode
METALINT=gometalinter --tests --disable-all --vendor --deadline=5m -e "zz_.*\.go" \
	 ./... --enable

test: vendor

lint: $(LINTERS)

$(LINTERS): vendor
	$(METALINT) $@

.PHONY: $(LINTERS) test lint

#################################################
# Create generated files
#################################################
GENERATED_FILES=$(API_VERSIONS:%=pkg/api/%/zz_generated.go)

api-versions:
	@echo $(API_VERSIONS)

$(GENERATED_FILES):
	deepcopy-gen -v=5 -h boilerplate.go.txt -i $(PKG)/$(patsubst %/zz_generated.go,%,$@) -O zz_generated

generated: $(GENERATED_FILES)

.PHONY: $(GENERATED_FILES)
