MAKEFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
PROJECT_PATH := $(dir $(patsubst %/,%,$(dir $(MAKEFILE_PATH))))
export GOPATH = $(PROJECT_PATH)

default: build

deps:
	go get github.com/coreos/bbolt

build: 
	go build src/tinynfsd.go
