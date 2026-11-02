package server

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/png"
	"strings"
	"testing"
)

// pngWithDimensions returns a valid 1×1 PNG whose IHDR claims w×h.
func pngWithDimensions(t *testing.T, w, h uint32) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewGray(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	b := buf.Bytes()
	// Signature (8) + length (4) + "IHDR" (4), then width, height.
	binary.BigEndian.PutUint32(b[16:], w)
	binary.BigEndian.PutUint32(b[20:], h)
	binary.BigEndian.PutUint32(b[29:], crc32.ChecksumIEEE(b[12:29]))
	return b
}

func TestProcessAvatarRejectsHugeDimensions(t *testing.T) {
	_, err := processAvatar(pngWithDimensions(t, 30000, 30000), false)
	if err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("err = %v, want dimensions too large", err)
	}
}

func TestProcessAvatarAcceptsNormalImage(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewGray(image.Rect(0, 0, 300, 200))); err != nil {
		t.Fatal(err)
	}
	if _, err := processAvatar(buf.Bytes(), false); err != nil {
		t.Fatalf("processAvatar: %v", err)
	}
}
