MAKEFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
PROJECT_PATH := $(patsubst %/,%,$(dir $(patsubst %/,%,$(dir $(MAKEFILE_PATH)))))
export GOPATH = $(PROJECT_PATH)

TIMESTAMP := $(shell /bin/date "+%s")

default: exec

exec:
	go run src/tinynfsd.go

deps:
	go get golang.org/x/image/draw
	go get github.com/etcd-io/bbolt

build: deps
	go build -ldflags="-s -w" -o bin/tinynfsd src/tinynfsd.go

archive: build
	tar cvzf tinynfs-archive-${TIMESTAMP}.tar.gz bin etc

clean:
	rm -f bin/tinynfsd
	rm -f tinynfs-archive-*

