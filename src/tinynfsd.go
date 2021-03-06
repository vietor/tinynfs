package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
	"tinynfs"
)

var (
	version = "1.1"
	command = struct {
		h bool
		t bool
		T bool
		c string
		d string
	}{}
)

func init() {
	flag.BoolVar(&command.h, "h", false, "this help")
	flag.BoolVar(&command.T, "T", false, "dump configuration")
	flag.BoolVar(&command.t, "t", false, "test configuration and exit")
	flag.StringVar(&command.c, "c", "etc/tinynfsd.conf", "set configuration `file`")
	flag.StringVar(&command.d, "d", "data/", "set data storage `path`")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "tinynfsd version: %s\n\nOptions:\n", version)
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	if command.h {
		flag.Usage()
		return
	}

	cfile, err := filepath.Abs(command.c)
	if err != nil {
		log.Fatalln(err)
	}
	dpath, err := filepath.Abs(command.d)
	if err != nil {
		log.Fatalln(err)
	}

	if command.t {
		config, err := tinynfs.NewConfig(cfile)
		if err != nil {
			fmt.Println(err)
			fmt.Printf("configuration file %s test failed\n", cfile)
		} else {
			fmt.Printf("configuration file %s is successful\n", cfile)
			if command.T {
				fmt.Println(config.Dump())
			}
		}
		return
	}

	config, err := tinynfs.NewConfig(cfile)
	if err != nil {
		log.Fatalln(err)
	} else if command.T {
		fmt.Println(config.Dump())
	}

	storage, err := tinynfs.NewFileSystem(dpath, config.Storage)
	if err != nil {
		log.Fatalln(err)
	}
	server, err := tinynfs.NewHttpServer(storage, config.Network)
	if err != nil {
		storage.Close()
		log.Fatalln(err)
	}

	ticker := time.NewTicker(time.Second * 30)
	go func() {
		for _ = range ticker.C {
			storage.Snapshot(false)
		}
	}()
	tinynfs.WaitProcessExit(func() {
		ticker.Stop()
		server.Close()
		storage.Close()
	})
}
