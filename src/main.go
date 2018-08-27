package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"time"
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
func main() {
	cwd := GetCWD()
	fmt.Println("cwd: " + cwd)
}
