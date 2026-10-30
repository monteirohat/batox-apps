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

// Countdown is a compact timer card: a small title, the time left in large
// digits and a progress bar. It turns amber near the end and red in the last
// seconds. Render it in a corner with overlay.Run, repainting every second:
//
//	t := ui.Countdown{Title: "Time left", Total: total, Align: ui.AlignRight}
//	overlay.Run(ctx, overlay.Options{Position: overlay.Top}, time.Second, func(w int) (*image.RGBA, error) {
//		t.Remaining = time.Until(end)
//		return t.Render(w)
//	})
//
// Its size depends only on Total and Title, so the card doesn't change size
// every second.
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

// stateColor is the color of the digits and accent: nil for the theme's own,
// amber near the end, red in the last seconds.
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
	if c.Theme.Accent == nil && c.Theme.Surface == (color.RGBA{}) {
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
	digitsFace, err := Face(Bold, size*1.5)
	if err != nil {
		return nil, err
	}
	defer digitsFace.Close()
	titleFace, err := Face(Regular, size*0.55)
	if err != nil {
		return nil, err
	}
	defer titleFace.Close()

	pad := int(size * 0.6)
	accent := int(math.Max(5, size*0.18))
	radius := int(size * 0.45)
	margin := int(size * 0.9)
	titleH := 0
	if c.Title != "" {
		titleH = int(size * 0.55 * 1.6)
	}
	digitsH := int(size * 1.5 * 1.2)
	barH, barGap := 0, 0
	if c.Total > 0 {
		barH, barGap = int(math.Max(4, size*0.16)), int(size*0.3)
	}

	textW := max(Measure(digitsFace, c.widthTemplate())*105/100, Measure(titleFace, c.Title))
	cardW := min(textW+2*pad+accent, screenWidth)
	cardH := 2*pad + titleH + digitsH + barGap + barH

	img := image.NewRGBA(image.Rect(0, 0, screenWidth, cardH+2*margin))
	x0, y0 := c.Align.x(screenWidth, cardW, margin), margin

	state := c.stateColor()
	accentAt := func(t float64) color.RGBA {
		if state != nil {
			return *state
		}
		return Gradient(c.Theme.Accent, t)
	}
	FillRoundedRect(img, image.Rect(x0, y0, x0+cardW, y0+cardH), radius, func(x, y int) color.RGBA {
		if x < accent {
			return accentAt(float64(y) / float64(cardH))
		}
		return c.Theme.Surface
	})

	textX := x0 + accent + pad
	y := y0 + pad
	if c.Title != "" {
		titleColor := c.Theme.Title
		if state != nil {
			titleColor = *state
		}
		DrawText(img, titleFace, titleColor, textX, y, c.Title)
		y += titleH
	}
	digitsColor := c.Theme.Text
	if state != nil {
		digitsColor = *state
	}
	DrawText(img, digitsFace, digitsColor, textX, y, FormatDuration(c.Remaining))
	y += digitsH

	if c.Total > 0 {
		y += barGap
		barW := cardW - accent - 2*pad
		track := image.Rect(textX, y, textX+barW, y+barH)
		FillRoundedRect(img, track, barH/2, func(int, int) color.RGBA { return color.RGBA{0x26, 0x34, 0x4F, 0xFF} })
		left := math.Max(0, math.Min(1, float64(c.Remaining)/float64(c.Total)))
		if fill := int(float64(barW) * left); fill > 0 {
			FillRoundedRect(img, image.Rect(textX, y, textX+fill, y+barH), barH/2, func(x, _ int) color.RGBA {
				return accentAt(float64(x) / float64(barW))
			})
		}
	}
	return img, nil
}
