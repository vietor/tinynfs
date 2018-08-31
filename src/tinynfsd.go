package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"tinynfs"
)

var (
	version = "1.0"
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

	plocker := tinynfs.NewProcessLock(filepath.Join(dpath, "tinynfsd.lock"))
	if err := plocker.Lock(); err != nil {
		log.Fatalln(err)
	}
	defer plocker.Unlock()

	storage, err := tinynfs.NewFileSystem(dpath, config.Storage)
	if err != nil {
		log.Fatalln(err)
	}
	server, err := tinynfs.NewHttpServer(storage, config.Network)
	if err != nil {
		storage.Close()
		log.Fatalln(err)
	}
	StartSignal(server, storage)
}

func StartSignal(server *tinynfs.HttpServer, storage *tinynfs.FileSystem) {
	var (
		sc chan os.Signal
		s  os.Signal
	)
	sc = make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT, syscall.SIGSTOP)
	for {
		s = <-sc
		switch s {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT, syscall.SIGSTOP:
			server.Close()
			storage.Close()
			return
		default:
			return
		}
	}
}
