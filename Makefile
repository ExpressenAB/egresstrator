SOURCEDIR=.
SOURCES := $(shell find $(SOURCEDIR) -name '*.go')

NAME=egresstrator

VERSION=0.0.1
BUILD_TIME=`date +%FT%T%z`

LDFLAGS=-ldflags "-X main.Version=${VERSION}"

all: build

build:
	@mkdir -p bin/
	go build ${LDFLAGS} -o bin/${NAME} egresstrator.go

xcompile:
	@rm -rf build/
	@mkdir -p build
	gox \
		-os="linux" \
		-os="windows" \
		-os="darwin" \
		-arch="amd64 386" \
		-output="build/{{.Dir}}_$(VERSION)_{{.OS}}_{{.Arch}}/$(NAME)"

package: xcompile
	$(eval FILES := $(shell ls build))
	@mkdir -p build/tgz
	for f in $(FILES); do \
		(cd $(shell pwd)/build && tar -zcvf tgz/$$f.tar.gz $$f); \
		echo $$f; \
	done

.PHONY: all build xcompile package

