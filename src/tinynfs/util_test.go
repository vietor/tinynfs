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
