package tinynfs

import (
	"encoding/base64"
	"testing"
)

var (
	tinyPNG, _ = base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVQYV2NgYAAAAAMAAWgmWQ0AAAAASUVORK5CYII=")
)

func TestImageParseBuffer(t *testing.T) {
	width, height, format, data, err := ImageParseBuffer(tinyPNG, 1000, 1000000)
	if err != nil {
		t.Error("ImageParseBuffer error", err)
	} else {
		t.Logf("ImageParseBuffer success: %d, %d, %s, %d", width, height, format, len(data))
	}
}

func TestImageScaleBuffer(t *testing.T) {
	width, height, format, data, err := ImageScaleBuffer(tinyPNG, 64, 64)
	if err != nil {
		t.Error("ImageScaleBuffer error", err)
	} else {
		t.Logf("ImageScaleBuffer success: %d, %d, %s, %d", width, height, format, len(data))
	}
}
