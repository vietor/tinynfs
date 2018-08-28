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

func TestDiskUsage(t *testing.T) {
	info, err := DiskUsage("../../test")
	if err != nil {
		t.Error("SysDiskUsage error", err)
	} else {
		t.Logf("SysDiskUsage success: %d, %d, %d", info.Size, info.Used, info.Free)
	}
}
