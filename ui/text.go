package ui

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// Weight selects one of the embedded fonts (Go Regular and Go Bold).
type Weight int

const (
	Regular Weight = iota
	Bold
)

var (
	fontsOnce sync.Once
	fonts     map[Weight]*opentype.Font
	fontsErr  error
)

func loadFonts() {
	fonts = map[Weight]*opentype.Font{}
	for w, ttf := range map[Weight][]byte{Regular: goregular.TTF, Bold: gobold.TTF} {
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
	return opentype.NewFace(fonts[w], &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
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
