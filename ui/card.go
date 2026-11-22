package ui

import (
	"errors"
	"image"
	"math"
)

// Card is a message card: a dark panel with square corners in a thin white
// frame, an optional label in small capitals and the text under it, wrapped to
// fit. Render it with an overlay:
//
//	card := ui.Card{Title: "BatoX", Text: "Dinner is ready!"}
//	overlay.Show(ctx, opts, card.Render)
type Card struct {
	Title string
	Text  string
	Theme Theme // zero value: BatoX

	// Scale multiplies the text size (default 1). The base size is about
	// 46 px on a 1920 px wide screen.
	Scale float64
	// MaxWidth is the widest the card may be, as a fraction of the screen
	// width (default 0.7).
	MaxWidth float64
	// MaxLines limits the text lines; the last one ends with "…" (default 4).
	MaxLines int
	// Align places the card on the screen (default centered).
	Align Align
}

func (c Card) withDefaults() Card {
	if c.Theme == (Theme{}) {
		c.Theme = BatoX
	}
	if c.Scale <= 0 {
		c.Scale = 1
	}
	if c.MaxWidth <= 0 || c.MaxWidth > 1 {
		c.MaxWidth = 0.7
	}
	if c.MaxLines <= 0 {
		c.MaxLines = 4
	}
	return c
}

// frameWidth is the width of a card's frame for a text size: about 4 px on a
// 1920 px wide screen, as the selection's frame on the console's home.
func frameWidth(size float64) int {
	return int(math.Max(2, math.Round(size*0.085)))
}

// Render draws the card centered in a transparent image as wide as the screen,
// with a margin above and below. It is an overlay.Painter.
func (c Card) Render(screenWidth int) (*image.RGBA, error) {
	if screenWidth <= 0 {
		return nil, errors.New("invalid screen width")
	}
	c = c.withDefaults()

	size := math.Max(22, float64(screenWidth)/42) * c.Scale
	textFace, err := Face(Regular, size)
	if err != nil {
		return nil, err
	}
	defer textFace.Close()
	labelFace, err := Face(Regular, size*0.42)
	if err != nil {
		return nil, err
	}
	defer labelFace.Close()

	label := Label(c.Title)
	spacing := int(math.Round(size * 0.42 * 0.07))
	frame := frameWidth(size)
	padX := int(size * 0.85)
	// The space above the label's capitals and under the last line's baseline.
	padY := int(size * 0.72)
	margin := int(size * 0.9)
	lineHeight := int(size * 1.3)
	m := textFace.Metrics()
	// A line's capitals start this far under its top, and its baseline is as
	// far above its bottom.
	inLine := (lineHeight-(m.Ascent.Ceil()+m.Descent.Ceil()))/2 + m.Ascent.Ceil() - m.CapHeight.Ceil()
	labelHeight := 0
	if label != "" {
		labelHeight = CapHeight(labelFace) + int(size*0.45) - inLine
	}

	maxCard := int(float64(screenWidth) * c.MaxWidth)
	lines := Wrap(textFace, c.Text, maxCard-2*padX-2*frame, c.MaxLines)

	widest := MeasureSpaced(labelFace, label, spacing)
	for _, line := range lines {
		widest = max(widest, Measure(textFace, line))
	}
	cardW := min(max(widest+2*padX+2*frame, int(size*8)), screenWidth)
	top := padY
	if label == "" {
		top = padY - inLine
	}
	cardH := 2*frame + top + labelHeight + len(lines)*lineHeight + padY - inLine

	img := image.NewRGBA(image.Rect(0, 0, screenWidth, cardH+2*margin))
	x0 := c.Align.x(screenWidth, cardW, margin)
	y0 := margin
	FillFramedRect(img, image.Rect(x0, y0, x0+cardW, y0+cardH), frame, c.Theme.Frame, c.Theme.Surface)

	textX := x0 + frame + padX
	y := y0 + frame + top
	if label != "" {
		DrawTextSpaced(img, labelFace, c.Theme.Title, textX, capTop(labelFace, y), spacing, label)
		y += labelHeight
	}
	lineTop := (lineHeight - (m.Ascent.Ceil() + m.Descent.Ceil())) / 2
	for _, line := range lines {
		DrawText(img, textFace, c.Theme.Text, textX, y+lineTop, line)
		y += lineHeight
	}
	return img, nil
}
