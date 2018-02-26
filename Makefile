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

vendor: Gopkg.lock
	dep ensure -v -vendor-only

update-vendor:
	dep ensure -v -update

.PHONY: $(BOOTSTRAP)

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

metalinter:
	gometalinter --install

lint: $(LINTERS)

$(LINTERS): metalinter vendor
	$(METALINT) $@

.PHONY: $(LINTERS) test lint

#################################################
# Create generated files
#################################################
GENERATED_FILES=$(API_VERSIONS:%=pkg/api/%/zz_generated.go)

deepcopy-gen:
	go get -u k8s.io/code-generator/cmd/deepcopy-gen

api-versions:
	@echo $(API_VERSIONS)

$(GENERATED_FILES):
	deepcopy-gen -v=5 -h boilerplate.go.txt -i $(PKG)/$(patsubst %/zz_generated.go,%,$@) -O zz_generated

generated: $(GENERATED_FILES)

.PHONY: $(GENERATED_FILES)

#################################################
# Building
#################################################
GOOS_OVERRIDE?=
PREFIX?=

GO_BUILD=CGO_ENABLED=0 go build -a -i
DOCKER_MAKE=GOOS_OVERRIDE='GOOS=linux' PREFIX=build/docker/$1/ make build/docker/$1/bin/$1

CMDs=$(sort $(patsubst cmd/%/,%,$(dir $(wildcard cmd/*/))))
BINS=$(addprefix bin/,$(CMDs))
DOCKER_IMAGES=$(addprefix docker-,$(CMDs))

$(CMDs:%=build/docker/%/Dockerfile):
	mkdir -p $(@D)
	cp Dockerfile.dev $@

$(BINS:%=$(PREFIX)%): $(PREFIX)bin/%: vendor
	$(GOOS_OVERRIDE) $(GO_BUILD) -o $@ $(patsubst $(PREFIX)bin/%,./cmd/%/...,$@)
$(BINS:%=%-dev):
	$(call DOCKER_MAKE,$(patsubst bin/%-dev,%,$@))
bins: $(BINS:%=$(PREFIX)%)

$(DOCKER_IMAGES):
	docker build -t manifoldco/heighliner:latest --build-arg BINARY=$(patsubst docker-%,bin/%,$@) .
$(DOCKER_IMAGES:%=%-dev): docker-%-dev: build/docker/%/Dockerfile bin/%-dev
	docker build -t manifoldco/heighliner:latest --build-arg BINARY=bin/$(patsubst docker-%-dev,%,$@) build/docker/$(patsubst docker-%-dev,%,$@)

docker: $(DOCKER_IMAGES)
docker-dev: $(DOCKER_IMAGES:%=%-dev)

.PHONY: $(BINS:%=$(PREFIX)%) $(DOCKER_IMAGES) $(CMDs:%=build/docker/%/Dockerfile)

#################################################
# Building the examples
#################################################
EXAMPLES=hello-world
DOCKER_EXAMPLES=$(addprefix docker-,$(EXAMPLES))

$(DOCKER_EXAMPLES):
	docker build -t hglnrio/$(patsubst docker-%,%,$@):latest _examples/$(patsubst docker-%,%,$@)

examples: $(DOCKER_EXAMPLES)

.PHONY: $(DOCKER_EXAMPLES)

#################################################
# Cleanup
#################################################
clean:
	rm -rf build
