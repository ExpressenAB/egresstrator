GOTOOLS=github.com/jteeuwen/go-bindata/... github.com/mitchellh/gox/...
SOURCEDIR := $(shell pwd)
SOURCES := $(shell find $(SOURCEDIR) -name '*.go')

NAME=egresstrator

VERSION=0.2.1
BUILD_TIME=`date +%FT%T%z`

LDFLAGS=-ldflags "-X main.Version=${VERSION}"

all: container build

container:
	@mkdir -p build/
	cd container && docker build . -t ${NAME}:latest && docker save -o ../build/${NAME}.tar ${NAME}:latest
	go-bindata -o container.go -prefix "build/" build/${NAME}.tar

build:
	@mkdir -p bin/
	go build ${LDFLAGS} -o bin/${NAME} ${NAME}.go container.go

xbuild: clean container
	@mkdir -p build
	gox \
		-os="linux" \
		-os="windows" \
		-os="darwin" \
		-arch="amd64 386" \
		${LDFLAGS} \
		-output="build/{{.Dir}}_$(VERSION)_{{.OS}}_{{.Arch}}/$(NAME)"

clean:
	@rm -rf build/ && rm -rf bin/ && rm -f container.go

package: xbuild
	$(eval FILES := $(shell ls build))
	@mkdir -p build/tgz
	for f in $(FILES); do \
		(cd $(shell pwd)/build && tar -zcvf tgz/$$f.tar.gz $$f); \
		echo $$f; \
	done

rpm:
	@mkdir -p build/rpm
	docker run --rm -it -v $(SOURCEDIR):/docker centos:7 /docker/package/rpm/build_rpm.sh ${VERSION}

tools:
	go get -u -v $(GOTOOLS)

ci: tools xbuild package rpm


.PHONY: all build xbuild package container

