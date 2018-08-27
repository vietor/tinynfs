package tinynfs

import (
	"path/filepath"
	"testing"
)

var (
	directTestBuffer = []byte("hello DirectStorage")
)

func TestDirectStorage(t *testing.T) {
	dstorage, _ := NewDirectStorage(filepath.Join("../../test", "data-directs"))
	filename, err := dstorage.WriteFile("", directTestBuffer)
	if err != nil {
		t.Error("WriteFile error", err)
	} else {
		t.Log("WriteFile success: " + filename)
	}
	data, err := dstorage.ReadFile(filename)
	if err != nil {
		t.Error("ReadFile error", err)
	} else {
		t.Log("ReadFile success: " + string(data))
	}
}
