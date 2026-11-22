package ui

import (
	"image"
	"image/color"
	"strings"
	"testing"
	"time"

	"golang.org/x/image/font/sfnt"
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

func TestFormatDuration(t *testing.T) {
	cases := map[time.Duration]string{
		0:                                     "0:00",
		-5 * time.Second:                      "0:00",
		500 * time.Millisecond:                "0:01", // rounds up: 0:00 means done
		59*time.Second + 200*time.Millisecond: "1:00",
		9*time.Minute + 5*time.Second:         "9:05",
		time.Hour + 2*time.Minute + 3*time.Second: "1:02:03",
	}
	for d, want := range cases {
		if got := FormatDuration(d); got != want {
			t.Errorf("FormatDuration(%v) = %q, want %q", d, got, want)
		}
	}
}

func TestCountdownSizeStaysTheSame(t *testing.T) {
	c := Countdown{Title: "Time left", Total: 30 * time.Minute, Align: AlignRight}
	sizeAt := func(left time.Duration) image.Rectangle {
		c.Remaining = left
		img, err := c.Render(1920)
		if err != nil {
			t.Fatal(err)
		}
		return img.Bounds()
	}
	first := sizeAt(30 * time.Minute)
	for _, left := range []time.Duration{10 * time.Minute, 9*time.Minute + 59*time.Second, 59 * time.Second, 5 * time.Second, 0} {
		if got := sizeAt(left); got != first {
			t.Errorf("at %v the image is %v, want %v (a size change every second would resize the overlay)", left, got, first)
		}
	}
	if first.Dx() != 1920 {
		t.Errorf("width = %d, want the screen width", first.Dx())
	}
}

func TestCountdownAlignment(t *testing.T) {
	opaque := func(img *image.RGBA, x, y int) bool { _, _, _, a := img.At(x, y).RGBA(); return a > 0 }
	for _, tc := range []struct {
		align       Align
		left, right bool // is the card near the left / right edge?
	}{
		{AlignLeft, true, false},
		{AlignRight, false, true},
	} {
		c := Countdown{Title: "Time left", Remaining: time.Minute, Total: time.Hour, Align: tc.align}
		img, err := c.Render(1920)
		if err != nil {
			t.Fatal(err)
		}
		mid := img.Bounds().Dy() / 2
		if got := opaque(img, 120, mid); got != tc.left {
			t.Errorf("align %v: near the left edge opaque = %v", tc.align, got)
		}
		if got := opaque(img, 1800, mid); got != tc.right {
			t.Errorf("align %v: near the right edge opaque = %v", tc.align, got)
		}
	}
}

func TestCountdownColors(t *testing.T) {
	c := Countdown{}.withDefaults()
	for left, want := range map[time.Duration]*color.RGBA{
		10 * time.Minute: nil,
		45 * time.Second: &Warning,
		5 * time.Second:  &Danger,
	} {
		c.Remaining = left
		if got := c.stateColor(); got != want {
			t.Errorf("at %v: color %v, want %v", left, got, want)
		}
	}
}

func TestCardHasSquareCornersAndAFrame(t *testing.T) {
	img, err := Card{Title: "BatoX", Text: "Dinner is ready!"}.Render(1920)
	if err != nil {
		t.Fatal(err)
	}
	// Find the card: the first and last opaque pixels of the middle row.
	b, mid := img.Bounds(), img.Bounds().Dy()/2
	left, right := -1, -1
	for x := 0; x < b.Dx(); x++ {
		if img.RGBAAt(x, mid).A > 0 {
			if left < 0 {
				left = x
			}
			right = x
		}
	}
	top := -1
	for y := 0; y < b.Dy() && top < 0; y++ {
		if img.RGBAAt((left+right)/2, y).A > 0 {
			top = y
		}
	}
	if left < 0 || top < 0 {
		t.Fatal("no card found")
	}
	for _, p := range []image.Point{{left, top}, {right, top}, {left, mid}, {(left + right) / 2, top}} {
		if got := img.RGBAAt(p.X, p.Y); got != BatoX.Frame {
			t.Errorf("at %v: %v, want the frame %v (square corners, a frame all around)", p, got, BatoX.Frame)
		}
	}
	if got := img.RGBAAt(left+12, mid); got == BatoX.Frame || got.A == 0 {
		t.Errorf("inside the frame: %v, want the surface", got)
	}
}

func TestDigitsDoNotShift(t *testing.T) {
	face, err := Face(Light, 60)
	if err != nil {
		t.Fatal(err)
	}
	defer face.Close()
	if a, b := MeasureDigits(face, "11:11"), MeasureDigits(face, "00:00"); a != b {
		t.Errorf("11:11 is %d px wide and 00:00 is %d: a countdown would shake", a, b)
	}
	if a, b := MeasureDigits(face, "1:00:00"), MeasureDigits(face, "00:00"); a <= b {
		t.Errorf("more digits should be wider: %d, %d", a, b)
	}
}

func TestLabelsAndWeights(t *testing.T) {
	if got := Label("Time left"); got != "TIME LEFT" {
		t.Errorf("Label = %q", got)
	}
	for _, w := range []Weight{Light, Regular, SemiBold, Bold} {
		face, err := Face(w, 20)
		if err != nil {
			t.Fatalf("weight %d: %v", w, err)
		}
		// The fonts cover what the earlier ones did: Latin with accents, Greek, Cyrillic.
		var buf sfnt.Buffer
		for _, r := range "Aãçé€ΩЖ…" {
			if i, err := fonts[w].GlyphIndex(&buf, r); err != nil || i == 0 {
				t.Errorf("weight %d has no glyph for %q", w, r)
			}
		}
		face.Close()
	}
	face, _ := Face(Regular, 20)
	defer face.Close()
	if plain, spaced := Measure(face, "ABC"), MeasureSpaced(face, "ABC", 3); spaced != plain+6 && spaced != plain+5 && spaced != plain+7 {
		t.Errorf("spaced width %d, plain %d: want 2 gaps of 3 px more", spaced, plain)
	}
}
