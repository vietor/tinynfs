package tinynfs

import (
	"path/filepath"
	"testing"
)

var (
	fsTestBuffer = []byte("hello FileSystem, ok")
)

func TestFileSystem(t *testing.T) {

	filename := "/a/a"
	fs, err := NewFileSystem(filepath.Join("../../test", "data-fs"), &Storage{
		DiskRemain:    4 * 1024 * 1024,
		VolumeMaxSize: 4 * 1024 * 1024 * 1024,
	})
	if err != nil {
		t.Error("Create", err)
	}
	err = fs.WriteFile(filename, "", "", fsTestBuffer)
	if err != nil {
		t.Error("Write file error", err)
	} else {
		t.Log("Write file success")
	}
	mime, metadata, data, err := fs.ReadFile(filename)
	if err != nil {
		t.Error("Read file error", err)
	} else {
		t.Log("Read file success:  " + mime + " " + metadata + " - " + string(data))
	}
}
