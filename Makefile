# ########################################################## #
# Makefile for Golang Project
# Includes cross-compiling, installation, cleanup

# credits to https://gist.github.com/cjbarker/5ce66fcca74a1928a155cfb3fea8fac4

# ########################################################## #

# Check for required command tools to build or stop immediately
EXECUTABLES = git go find pwd
K := $(foreach exec,$(EXECUTABLES),\
        $(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH")))

ROOT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

VERSION=1.0-beta1
BUILD=`git rev-parse HEAD`
PLATFORMS=darwin linux
ARCHITECTURES=amd64

MANAGER_BINARY=backr-manager
MANAGER_PACKAGE=github.com/agence-webup/backr/manager/bin/manager
# Setup linker flags option for build that interoperate with variable names in src code
MANAGER_LDFLAGS=-ldflags "-s -w -X ${MANAGER_PACKAGE}/cmd.version=${VERSION} -X ${MANAGER_PACKAGE}/cmd.build=${BUILD}"

default: build

all: clean build_all

build:
	go build ${MANAGER_LDFLAGS} -o ${MANAGER_BINARY} ${MANAGER_PACKAGE}

build_all:
	$(foreach GOOS, $(PLATFORMS),\
	$(foreach GOARCH, $(ARCHITECTURES), $(shell export GOOS=$(GOOS); export GOARCH=$(GOARCH); go build ${MANAGER_LDFLAGS} -v -o $(MANAGER_BINARY)-$(GOOS)-$(GOARCH) ${MANAGER_PACKAGE})))

# Remove only what we've created
clean:
	find ${ROOT_DIR} -name '${MANAGER_BINARY}[-?][a-zA-Z0-9]*[-?][a-zA-Z0-9]*' -delete

.PHONY: check clean build_all all
