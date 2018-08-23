package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"time"
	"tinynfs"
)

var (
	testBuffer = []byte("hello\ngo\n")
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

func testDirectStorage(cwd string) {
	dstorage := tinynfs.NewDirectStorage(filepath.Join(cwd, "data", "directs"))
	filename, err := dstorage.WriteFile(testBuffer)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("filename: " + filename)
	}
	data, err := dstorage.ReadFile(filename)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("read: " + string(data))
	}
}

func testVolumeStorage(cwd string) {
	bstorage := tinynfs.NewVolumeStorage(filepath.Join(cwd, "data", "volumes"))
	err := bstorage.WriteFile(0, 1000, testBuffer)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("write success: ")
	}
	data, err := bstorage.ReadFile(0, 1000, len(testBuffer))
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("read: " + string(data))
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	cwd := GetCWD()
	fmt.Println("cwd: " + cwd)
	//testDirectStorage(cwd)
	testVolumeStorage(cwd)
}
