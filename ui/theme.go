// Package ui draws the visual pieces of BatoX overlays: a theme, fonts with
// text wrapping, shapes and ready-made components such as Card.
//
// The look is the one of the BatoX web app and of batox-theme: a flat dark
// indigo surface in a thin white frame, square corners, white type, labels in
// small capitals over their values.
//
// Everything draws into *image.RGBA, which is premultiplied, as the overlay
// package expects. Sizes are derived from the screen width, so a component
// looks the same at 1080p and 4K.
package ui

import "image/color"

// Theme holds the colors of a component.
type Theme struct {
	Surface color.RGBA // card background; its alpha sets the transparency
	Frame   color.RGBA // the thin frame around the card
	Text    color.RGBA // main text and figures
	Title   color.RGBA // the label above the text, in small capitals
	Track   color.RGBA // the part of a progress line still to go
}

// BatoX is the default theme: the surface, frame and type of the BatoX web
// app. The surface is almost opaque, so a card reads over any game.
var BatoX = Theme{
	Surface: color.RGBA{0x15, 0x12, 0x2B, 0xF0},
	Frame:   color.RGBA{0xF2, 0xF4, 0xF8, 0xFF},
	Text:    color.RGBA{0xFF, 0xFF, 0xFF, 0xFF},
	Title:   color.RGBA{0xA9, 0xB7, 0xCC, 0xFF},
	Track:   color.RGBA{0xFF, 0xFF, 0xFF, 0x30},
}

// Status colors, for warnings and urgent states.
var (
	Warning = color.RGBA{0xFF, 0xBF, 0x4C, 0xFF} // amber
	Danger  = color.RGBA{0xFC, 0x6C, 0x69, 0xFF} // red
)

// Gradient returns the color at t (0 to 1) along stops, evenly spaced.
func Gradient(stops []color.RGBA, t float64) color.RGBA {
	if len(stops) == 0 {
		return color.RGBA{}
	}
	if len(stops) == 1 {
		return stops[0]
	}
	t = clamp01(t) * float64(len(stops)-1)
	i := int(t)
	if i >= len(stops)-1 {
		return stops[len(stops)-1]
	}
	f := t - float64(i)
	a, b := stops[i], stops[i+1]
	mix := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*f + 0.5) }
	return color.RGBA{mix(a.R, b.R), mix(a.G, b.G), mix(a.B, b.B), mix(a.A, b.A)}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
