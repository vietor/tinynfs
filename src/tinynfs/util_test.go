package tinynfs

import (
	"testing"
)

func TestGetPathDiskStat(t *testing.T) {
	info, err := GetPathDiskStat("../../test")
	if err != nil {
		t.Error("GetPathDiskStat error", err)
	} else {
		t.Logf("GetPathDiskStat success: %d, %d, %d", info.Size, info.Used, info.Free)
	}
}

func TestProcessLock(t *testing.T) {
	plock := NewProcessLock("../../test/test.lock")
	ok, err := plock.Lock()
	if err != nil {
		t.Error("ProcessLock error", err)
	} else if !ok {
		t.Error("ProcessLock failed")
	} else {
		t.Log("ProcessLock success")
	}
	plock.Unlock()
}
