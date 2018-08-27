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
	dstorage, _ := tinynfs.NewDirectStorage(filepath.Join(cwd, "data", "directs"))
	filename, err := dstorage.WriteFile("", testBuffer)
	if err != nil {
		fmt.Println(err)
		return
	} else {
		fmt.Println("filename: " + filename)
	}
	data, err := dstorage.ReadFile(filename)
	if err != nil {
		fmt.Println(err)
		return
	} else {
		fmt.Println("read: " + string(data))
	}
}

func testVolumeStorage(cwd string) {
	bstorage, _ := tinynfs.NewVolumeStorage(filepath.Join(cwd, "data", "volumes"), int64(len(testBuffer))+1)
	id, offset, err := bstorage.WriteFile(testBuffer)
	if err != nil {
		fmt.Println(err)
		return
	} else {
		fmt.Println("write success: ")
	}
	data, err := bstorage.ReadFile(id, offset, len(testBuffer))
	if err != nil {
		fmt.Println(err)
		return
	} else {
		fmt.Println("read: " + string(data))
	}
}

func testFileSystem(cwd string) {
	filename := "/a/a"
	fs, err := tinynfs.NewFileSystem(filepath.Join(cwd, "data"))
	if err != nil {
		fmt.Println(err)
		return
	}
	err = fs.WriteFile(filename, testBuffer)
	if err != nil {
		fmt.Println(err)
		return
	}
	data, err := fs.ReadFile(filename)
	if err != nil {
		fmt.Println(err)
		return
	} else {
		fmt.Println("read: " + string(data))
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	cwd := GetCWD()
	fmt.Println("cwd: " + cwd)
	//testDirectStorage(cwd)
	//testVolumeStorage(cwd)
	testFileSystem(cwd)
}
