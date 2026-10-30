package overlay

import (
	"image"
	"image/color"
	"testing"
)

func TestParsePosition(t *testing.T) {
	for s, want := range map[string]Position{"top": Top, "center": Center, "bottom": Bottom} {
		if got, err := ParsePosition(s); err != nil || got != want {
			t.Errorf("ParsePosition(%q) = %v, %v", s, got, err)
		}
	}
	if _, err := ParsePosition("middle"); err == nil {
		t.Error("expected an error for an unknown position")
	}
}

func TestBlitConvertsAndCenters(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	img.SetRGBA(0, 0, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	img.SetRGBA(1, 0, color.RGBA{R: 1, G: 2, B: 3, A: 4})

	dst := make([]byte, 4*4) // a 4x1 buffer: the 2px image goes in the middle
	for i := range dst {
		dst[i] = 0xEE // leftovers from an earlier frame must be cleared
	}
	blitARGB8888(img, dst, 4, 1)

	want := []byte{
		0, 0, 0, 0, // transparent left margin
		30, 20, 10, 255, // B, G, R, A
		3, 2, 1, 4,
		0, 0, 0, 0,
	}
	for i := range want {
		if dst[i] != want[i] {
			t.Fatalf("dst = %v, want %v", dst, want)
		}
	}
}

func TestBlitClipsWideImages(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 3))
	dst := make([]byte, 4*2*2)   // 2x2 buffer
	blitARGB8888(img, dst, 2, 2) // must not panic
}
