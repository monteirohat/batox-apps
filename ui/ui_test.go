package ui

import (
	"image/color"
	"strings"
	"testing"
)

func TestGradient(t *testing.T) {
	stops := []color.RGBA{{0, 0, 0, 255}, {200, 100, 50, 255}}
	if got := Gradient(stops, 0); got != stops[0] {
		t.Errorf("t=0: %v", got)
	}
	if got := Gradient(stops, 1); got != stops[1] {
		t.Errorf("t=1: %v", got)
	}
	if got := Gradient(stops, 0.5); got.R != 100 || got.G != 50 || got.B != 25 {
		t.Errorf("t=0.5: %v", got)
	}
	if got := Gradient(stops, 7); got != stops[1] {
		t.Errorf("t past the end should clamp, got %v", got)
	}
}

func TestWrap(t *testing.T) {
	face, err := Face(Bold, 40)
	if err != nil {
		t.Fatal(err)
	}
	maxWidth := Measure(face, "Dinner is ready")
	lines := Wrap(face, "Dinner is ready, come to the table now please", maxWidth, 0)
	if len(lines) < 2 {
		t.Fatalf("expected several lines, got %q", lines)
	}
	for _, l := range lines {
		if Measure(face, l) > maxWidth {
			t.Errorf("line %q is wider than the limit", l)
		}
	}

	if got := Wrap(face, "One\nTwo", 1000, 0); len(got) != 2 || got[1] != "Two" {
		t.Errorf("explicit line breaks: %q", got)
	}

	long := Wrap(face, "a\nb\nc\nd\ne\nf", 1000, 4)
	if len(long) != 4 || !strings.HasSuffix(long[3], "…") {
		t.Errorf("max lines: %q", long)
	}
}

func TestCardRender(t *testing.T) {
	card := Card{Title: "BatoX", Text: "Dinner is ready!"}
	img, err := card.Render(1920)
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	if b.Dx() != 1920 {
		t.Errorf("width = %d, want the screen width", b.Dx())
	}
	if b.Dy() < 100 || b.Dy() > 600 {
		t.Errorf("height = %d, looks wrong for one line", b.Dy())
	}
	if _, _, _, a := img.At(0, 0).RGBA(); a != 0 {
		t.Error("the corner outside the card should be transparent")
	}
	if _, _, _, a := img.At(b.Dx()/2, b.Dy()/2).RGBA(); a == 0 {
		t.Error("the middle of the card should be opaque")
	}

	// Sizes follow the screen width.
	big, err := card.Render(3840)
	if err != nil {
		t.Fatal(err)
	}
	if big.Bounds().Dy() <= b.Dy() {
		t.Errorf("a 4K card (%d px) should be taller than a 1080p one (%d px)", big.Bounds().Dy(), b.Dy())
	}

	if _, err := card.Render(0); err == nil {
		t.Error("expected an error for a zero width")
	}
}
