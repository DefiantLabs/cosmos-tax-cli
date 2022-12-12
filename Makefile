#!/usr/bin/make -f

BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
COMMIT := $(shell git log -1 --format='%H')

# don't override user values
ifeq (,$(VERSION))
  VERSION := $(shell git describe --tags)
  # if VERSION is empty, then populate it with branch's name and raw commit hash
  ifeq (,$(VERSION))
    VERSION := $(BRANCH)-$(COMMIT)
  endif
endif

FQCN = ghcr.io/defiantlabs/cosmos-tax-cli-private/cosmos-tax-cli-private# default value, overide with: make -e FQCN="foo"

all: install

install: go.sum
	go install .

build:
	go build -o bin/cosmos-tax-cli-private .

clean:
	rm -rf build

build-docker:
	docker build -t $(FQCN):$(VERSION) -f ./Dockerfile .
	docker push $(FQCN):$(VERSION)