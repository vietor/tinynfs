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
		DiskRemain:       4 * 1024 * 1024,
		SnapshotInterval: 600,
		SnapshotReserve:  3,
		VolumeSliceSize:  4 * 1024 * 1024 * 1024,
		VolumeFileGroups: []VolumeGroup{
			VolumeGroup{
				Id:   1,
				Path: "{{DATA}}/volumes1/",
			},
			VolumeGroup{
				Id:   0,
				Path: "{{DATA}}/volumes/",
			},
		},
	})
	if err != nil {
		t.Error("Create", err)
	}
	err = fs.WriteFile(filename, "", "", fsTestBuffer, nil)
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
	ssfile, err := fs.Snapshot(true)
	if err != nil {
		t.Error("Snapshot error", err)
	} else {
		t.Log("Snapshot file success: " + ssfile)
	}
}
