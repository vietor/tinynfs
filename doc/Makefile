MAKEFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
GOPATH := $(dir $(patsubst %/,%,$(dir $(MAKEFILE_PATH))))

default: build

deps:
	go get github.com/coreos/bbolt

build: deps
	go build
