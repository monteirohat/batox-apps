package ui

import (
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// Inter (SIL Open Font License, fonts/Inter-LICENSE.txt), cut down to the
// Latin, Greek and Cyrillic letters, so the apps don't depend on the fonts
// installed on the console.
var (
	//go:embed fonts/Inter-Light.ttf
	interLight []byte
	//go:embed fonts/Inter-Regular.ttf
	interRegular []byte
	//go:embed fonts/Inter-SemiBold.ttf
	interSemiBold []byte
)

// Weight selects one of the embedded fonts: Inter Light for large figures,
// Regular for text and labels, SemiBold for emphasis.
type Weight int

const (
	Regular Weight = iota
	SemiBold
	Light

	// Bold is the heaviest embedded weight, kept for programs written for the
	// earlier fonts.
	Bold = SemiBold
)

var (
	fontsOnce sync.Once
	fonts     map[Weight]*opentype.Font
	fontsErr  error
)

func loadFonts() {
	fonts = map[Weight]*opentype.Font{}
	for w, ttf := range map[Weight][]byte{Light: interLight, Regular: interRegular, SemiBold: interSemiBold} {
		f, err := opentype.Parse(ttf)
		if err != nil {
			fontsErr = fmt.Errorf("loading the embedded fonts: %w", err)
			return
		}
		fonts[w] = f
	}
}

// Face returns an embedded font at size pixels.
func Face(w Weight, size float64) (font.Face, error) {
	fontsOnce.Do(loadFonts)
	if fontsErr != nil {
		return nil, fontsErr
	}
	f, ok := fonts[w]
	if !ok {
		return nil, fmt.Errorf("unknown font weight %d", w)
	}
	return opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingNone})
}

// Measure returns the width of s in pixels.
func Measure(face font.Face, s string) int {
	return font.MeasureString(face, s).Ceil()
}

// Wrap splits text into lines no wider than maxWidth. Explicit line breaks are
// kept; past maxLines (0 for no limit) the last line ends with "…".
func Wrap(face font.Face, text string, maxWidth, maxLines int) []string {
	var lines []string
	for _, para := range strings.Split(text, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}
		line := words[0]
		for _, word := range words[1:] {
			if Measure(face, line+" "+word) <= maxWidth {
				line += " " + word
				continue
			}
			lines = append(lines, line)
			line = word
		}
		lines = append(lines, line)
	}
	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[:maxLines]
		last := []rune(lines[maxLines-1])
		for len(last) > 0 && Measure(face, string(last)+"…") > maxWidth {
			last = last[:len(last)-1]
		}
		lines[maxLines-1] = strings.TrimSpace(string(last)) + "…"
	}
	return lines
}

// DrawText draws s with its top-left corner at (x, top).
func DrawText(img *image.RGBA, face font.Face, c color.RGBA, x, top int, s string) {
	d := font.Drawer{Dst: img, Src: image.NewUniform(c), Face: face}
	d.Dot = fixed.P(x, top+face.Metrics().Ascent.Ceil())
	d.DrawString(s)
}

// CapHeight is the height of the face's capital letters and digits, in pixels.
func CapHeight(face font.Face) int {
	return face.Metrics().CapHeight.Ceil()
}

// capTop is where to put the top of a line of text for its capitals to start
// at y.
func capTop(face font.Face, y int) int {
	m := face.Metrics()
	return y - (m.Ascent.Ceil() - m.CapHeight.Ceil())
}

// Label is how a label is written: in capitals.
func Label(s string) string {
	return strings.ToUpper(s)
}

// MeasureSpaced returns the width of s drawn by DrawTextSpaced.
func MeasureSpaced(face font.Face, s string, spacing int) int {
	n := len([]rune(s))
	if n == 0 {
		return 0
	}
	w := fixed.Int26_6(0)
	for _, r := range s {
		a, _ := face.GlyphAdvance(r)
		w += a
	}
	return w.Ceil() + spacing*(n-1)
}

// DrawTextSpaced draws s with its top-left corner at (x, top) and spacing
// extra pixels between letters, as labels in capitals are set.
func DrawTextSpaced(img *image.RGBA, face font.Face, c color.RGBA, x, top, spacing int, s string) {
	d := font.Drawer{Dst: img, Src: image.NewUniform(c), Face: face}
	d.Dot = fixed.P(x, top+face.Metrics().Ascent.Ceil())
	for _, r := range s {
		d.DrawString(string(r))
		d.Dot.X += fixed.I(spacing)
	}
}

// digitCell is the width every digit takes in DrawDigits: the widest one's.
func digitCell(face font.Face) fixed.Int26_6 {
	cell := fixed.Int26_6(0)
	for r := '0'; r <= '9'; r++ {
		if a, ok := face.GlyphAdvance(r); ok && a > cell {
			cell = a
		}
	}
	return cell
}

// MeasureDigits returns the width of s drawn by DrawDigits. It depends on how
// many digits s has, not on which ones.
func MeasureDigits(face font.Face, s string) int {
	cell, w := digitCell(face), fixed.Int26_6(0)
	for _, r := range s {
		if r >= '0' && r <= '9' {
			w += cell
		} else if a, ok := face.GlyphAdvance(r); ok {
			w += a
		}
	}
	return w.Ceil()
}

// DrawDigits draws a number such as "12:05" with its top-left corner at
// (x, top), every digit centered in a cell of the same width, so the number
// doesn't shift as its digits change.
func DrawDigits(img *image.RGBA, face font.Face, c color.RGBA, x, top int, s string) {
	cell := digitCell(face)
	d := font.Drawer{Dst: img, Src: image.NewUniform(c), Face: face}
	pen := fixed.I(x)
	baseline := fixed.I(top + face.Metrics().Ascent.Ceil())
	for _, r := range s {
		a, _ := face.GlyphAdvance(r)
		w := a
		if r >= '0' && r <= '9' {
			w = cell
		}
		d.Dot = fixed.Point26_6{X: pen + (w-a)/2, Y: baseline}
		d.DrawString(string(r))
		pen += w
	}
}
