package tinynfsd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"tinynfs"
)

func GetCWD() string {
	cwd := os.Getenv("GOPATH")
	if cwd == "" {
		efile, _ := exec.LookPath(os.Args[0])
		epath := filepath.Dir(filepath.Dir(efile))
		cwd, _ = filepath.Abs(epath)
	}
	return cwd
}

func StartSignal(server *tinynfs.HttpServer) {
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
			return
		default:
			return
		}
	}
}

func main() {
	cwd := GetCWD()
	fs, err := tinynfs.NewFileSystem(filepath.Join(cwd, "data"))
	if err != nil {
		fmt.Println(err)
		return
	}
	server, err := tinynfs.NewHttpServer(fs, ":8090")
	if err != nil {
		fmt.Println(err)
		return
	}
	StartSignal(server)
}