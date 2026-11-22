package ui

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"time"
)

// Countdown is a compact timer card: a label in small capitals, the time left
// in large light digits and a thin progress line, in a dark panel with a white
// frame. It turns amber near the end and red in the last seconds. Render it in
// a corner with overlay.Run, repainting every second:
//
//	t := ui.Countdown{Title: "Time left", Total: total, Align: ui.AlignRight}
//	overlay.Run(ctx, overlay.Options{Position: overlay.Top}, time.Second, func(w int) (*image.RGBA, error) {
//		t.Remaining = time.Until(end)
//		return t.Render(w)
//	})
//
// Its size depends only on Total and Title, and every digit takes the same
// width, so the card doesn't change size or shake every second.
type Countdown struct {
	Title     string        // small line above the digits, e.g. "Time left"
	Remaining time.Duration // time left; 0 or less shows 0:00
	Total     time.Duration // the whole countdown, for the progress bar; 0 hides the bar
	Align     Align
	Theme     Theme   // zero value: BatoX
	Scale     float64 // size multiplier (default 1)

	// WarnAt and UrgentAt are when the timer turns amber and red
	// (defaults: 1 minute and 10 seconds).
	WarnAt, UrgentAt time.Duration
}

// FormatDuration shows a duration as M:SS, or H:MM:SS from one hour on. It
// rounds up, so 0:00 only shows when the time is really over.
func FormatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	secs := int64((d + time.Second - 1) / time.Second)
	h, m, s := secs/3600, secs%3600/60, secs%60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// widthTemplate is the widest the digits get during the countdown: every
// digit replaced by "0", for the longest format the countdown reaches.
func (c Countdown) widthTemplate() string {
	longest := max(c.Total, c.Remaining)
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return '0'
		}
		return r
	}, FormatDuration(longest))
}

// stateColor is the color of the digits, the line and the frame: nil for the
// theme's own, amber near the end, red in the last seconds.
func (c Countdown) stateColor() *color.RGBA {
	switch {
	case c.Remaining <= c.UrgentAt:
		return &Danger
	case c.Remaining <= c.WarnAt:
		return &Warning
	}
	return nil
}

func (c Countdown) withDefaults() Countdown {
	if c.Theme == (Theme{}) {
		c.Theme = BatoX
	}
	if c.Scale <= 0 {
		c.Scale = 1
	}
	if c.WarnAt <= 0 {
		c.WarnAt = time.Minute
	}
	if c.UrgentAt <= 0 {
		c.UrgentAt = 10 * time.Second
	}
	return c
}

// Render draws the timer in a transparent image as wide as the screen, aligned
// as asked. It is an overlay.Painter.
func (c Countdown) Render(screenWidth int) (*image.RGBA, error) {
	if screenWidth <= 0 {
		return nil, errors.New("invalid screen width")
	}
	c = c.withDefaults()

	size := math.Max(18, float64(screenWidth)/60) * c.Scale // ~32 px at 1920
	digitsFace, err := Face(Light, size*1.9)
	if err != nil {
		return nil, err
	}
	defer digitsFace.Close()
	labelFace, err := Face(Regular, size*0.5)
	if err != nil {
		return nil, err
	}
	defer labelFace.Close()

	label := Label(c.Title)
	spacing := int(math.Round(size * 0.5 * 0.07))
	frame := frameWidth(size * 1.2)
	pad := int(size * 0.7)
	margin := int(size * 0.9)
	labelH := 0
	if label != "" {
		labelH = CapHeight(labelFace) + int(size*0.42)
	}
	digitsH := CapHeight(digitsFace)
	barH, barGap := 0, 0
	if c.Total > 0 {
		barH, barGap = int(math.Max(2, math.Round(size*0.1))), int(size*0.5)
	}

	textW := max(MeasureDigits(digitsFace, c.widthTemplate()), MeasureSpaced(labelFace, label, spacing))
	cardW := min(textW+2*pad+2*frame, screenWidth)
	cardH := 2*frame + 2*pad + labelH + digitsH + barGap + barH

	img := image.NewRGBA(image.Rect(0, 0, screenWidth, cardH+2*margin))
	x0, y0 := c.Align.x(screenWidth, cardW, margin), margin

	// Near the end the figures, the line and the frame take the state's color.
	ink, frameColor := c.Theme.Text, c.Theme.Frame
	if state := c.stateColor(); state != nil {
		ink, frameColor = *state, *state
	}
	FillFramedRect(img, image.Rect(x0, y0, x0+cardW, y0+cardH), frame, frameColor, c.Theme.Surface)

	textX := x0 + frame + pad
	y := y0 + frame + pad
	if label != "" {
		DrawTextSpaced(img, labelFace, c.Theme.Title, textX, capTop(labelFace, y), spacing, label)
		y += labelH
	}
	DrawDigits(img, digitsFace, ink, textX, capTop(digitsFace, y), FormatDuration(c.Remaining))
	y += digitsH

	if c.Total > 0 {
		y += barGap
		barW := cardW - 2*pad - 2*frame
		FillRect(img, image.Rect(textX, y, textX+barW, y+barH), c.Theme.Track)
		left := math.Max(0, math.Min(1, float64(c.Remaining)/float64(c.Total)))
		if fill := int(float64(barW) * left); fill > 0 {
			FillRect(img, image.Rect(textX, y, textX+fill, y+barH), ink)
		}
	}
	return img, nil
}
