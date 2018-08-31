MAKEFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
PROJECT_PATH := $(patsubst %/,%,$(dir $(patsubst %/,%,$(dir $(MAKEFILE_PATH)))))
export GOPATH = $(PROJECT_PATH)

default: exec

deps:
	go get golang.org/x/image/draw
	go get github.com/etcd-io/bbolt

exec:
	go run src/tinynfsd.go

build:
	go build -ldflags="-s -w" -o bin/tinynfsd src/tinynfsd.go

