package tinynfs

import (
	"testing"
)

func TestRandHex(t *testing.T) {
	hex := RandHex(5)
	if len(hex) != 10 {
		t.Error("RandomHex error")
	} else {
		t.Log("RandomHex success")
	}
}

func TestGetDiskStat(t *testing.T) {
	info, err := GetDiskStat("../../test")
	if err != nil {
		t.Error("GetDiskStat error", err)
	} else {
		t.Logf("GetDiskStat success: %d, %d, %d", info.Size, info.Used, info.Free)
	}
}

func TestFileLock(t *testing.T) {
	lock := NewFileLock("../../test/test.lock")
	err := lock.Lock()
	if err != nil {
		t.Error("File lock error", err)
	} else {
		t.Logf("File lock success")
	}
	lock.Unlock()
}
