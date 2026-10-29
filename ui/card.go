package ui

import (
	"errors"
	"image"
	"image/color"
	"math"
)

// Card is a message card: a dark rounded panel with a gradient accent bar on
// the left, an optional small title and the text, wrapped to fit. Render it
// with an overlay:
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
}

func (c Card) withDefaults() Card {
	if c.Theme.Accent == nil && c.Theme.Surface == (color.RGBA{}) {
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

// Render draws the card centered in a transparent image as wide as the screen,
// with a margin above and below. It is an overlay.Painter.
func (c Card) Render(screenWidth int) (*image.RGBA, error) {
	if screenWidth <= 0 {
		return nil, errors.New("invalid screen width")
	}
	c = c.withDefaults()

	size := math.Max(22, float64(screenWidth)/42) * c.Scale
	textFace, err := Face(Bold, size)
	if err != nil {
		return nil, err
	}
	defer textFace.Close()
	titleFace, err := Face(Regular, size*0.55)
	if err != nil {
		return nil, err
	}
	defer titleFace.Close()

	pad := int(size * 0.8)
	accent := int(math.Max(6, size*0.22))
	radius := int(size * 0.5)
	margin := int(size * 0.9)
	lineHeight := int(size * 1.3)
	titleHeight := 0
	if c.Title != "" {
		titleHeight = int(size * 0.55 * 1.6)
	}

	maxCard := int(float64(screenWidth) * c.MaxWidth)
	lines := Wrap(textFace, c.Text, maxCard-2*pad-accent, c.MaxLines)

	widest := Measure(titleFace, c.Title)
	for _, line := range lines {
		widest = max(widest, Measure(textFace, line))
	}
	cardW := min(max(widest+2*pad+accent, int(size*8)), screenWidth)
	cardH := 2*pad + titleHeight + len(lines)*lineHeight

	img := image.NewRGBA(image.Rect(0, 0, screenWidth, cardH+2*margin))
	x0 := (screenWidth - cardW) / 2
	y0 := margin
	FillRoundedRect(img, image.Rect(x0, y0, x0+cardW, y0+cardH), radius, func(x, y int) color.RGBA {
		if x < accent {
			return Gradient(c.Theme.Accent, float64(y)/float64(cardH))
		}
		return c.Theme.Surface
	})

	textX := x0 + accent + pad
	y := y0 + pad
	if c.Title != "" {
		DrawText(img, titleFace, c.Theme.Title, textX, y, c.Title)
		y += titleHeight
	}
	for _, line := range lines {
		DrawText(img, textFace, c.Theme.Text, textX, y, line)
		y += lineHeight
	}
	return img, nil
}
