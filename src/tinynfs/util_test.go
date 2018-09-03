package tinynfs

import (
	"testing"
)

func TestTimeHex(t *testing.T) {
	hex := TimeHex(0)
	t.Log("Timehex success, " + hex)
	hex = TimeHex(1)
	t.Log("Timehex success, " + hex)
}

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
