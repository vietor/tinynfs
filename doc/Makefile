MAKEFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
PROJECT_PATH := $(dir $(patsubst %/,%,$(dir $(MAKEFILE_PATH))))
export GOPATH = $(PROJECT_PATH)

default: exec

deps:
	go get github.com/coreos/bbolt

build: 
	go build src/tinynfsd.go

exec:
	go run src/tinynfsd.go

