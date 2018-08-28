package tinynfs

import (
	"path/filepath"
	"testing"
)

var (
	fsSmallTestBuffer  = []byte("hello FileSystem small")
	fsBiggerTestBuffer = []byte("hello FileSystem bigger")
)

func TestFileSystem(t *testing.T) {

	filename := "/a/a"
	fs, err := NewFileSystem(filepath.Join("../../test", "data-fs"), &Storage{
		DirectMinSize: 4 * 1024 * 1024,
		VolumeMaxSize: 4 * 1024 * 1024 * 1024,
	})
	if err != nil {
		t.Error("Create", err)
	}
	err = fs.WriteFile(filename, "", fsSmallTestBuffer)
	if err != nil {
		t.Error("Write small file error", err)
	} else {
		t.Log("Write small file success")
	}
	mime, data, err := fs.ReadFile(filename)
	if err != nil {
		t.Error("Read small file error", err)
	} else {
		t.Log("Read small file success:  " + mime + " - " + string(data))
	}
}
