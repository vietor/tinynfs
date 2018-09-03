package tinynfs

import (
	"fmt"
	"path/filepath"
	"testing"
)

var (
	volumeTestBuffer = []byte("hello VolumeStorage")
)

func TestVolumeStorage(t *testing.T) {
	bstorage, _ := NewVolumeStorage(filepath.Join("../../test", "data-volumes"), int64(len(volumeTestBuffer)+1), 50*1024)
	id, offset, err := bstorage.WriteFile(volumeTestBuffer)
	if err != nil {
		t.Error("WriteFile error", err)
	} else {
		t.Log(fmt.Sprintf("WriteFile success: %d %d", id, offset))
	}
	data, err := bstorage.ReadFile(id, offset, len(volumeTestBuffer))
	if err != nil {
		t.Error("ReadFile error", err)
	} else {
		t.Log("ReadFile success: " + string(data))
	}
}
