// Package ui draws the visual pieces of BatoX overlays: a theme, fonts with
// text wrapping, rounded shapes and ready-made components such as Card.
//
// Everything draws into *image.RGBA, which is premultiplied, as the overlay
// package expects. Sizes are derived from the screen width, so a component
// looks the same at 1080p and 4K.
package ui

import "image/color"

// Theme holds the colors of a component.
type Theme struct {
	Surface color.RGBA   // card background; its alpha sets the transparency
	Text    color.RGBA   // main text
	Title   color.RGBA   // small title above the text
	Accent  []color.RGBA // gradient of the accent bar, top to bottom
}

// BatoX is the default theme, with the colors of the BatoX logo.
var BatoX = Theme{
	Surface: color.RGBA{0x12, 0x1A, 0x2C, 0xF0},
	Text:    color.RGBA{0xE9, 0xEE, 0xF8, 0xFF},
	Title:   color.RGBA{0x6A, 0xA8, 0xFF, 0xFF},
	Accent: []color.RGBA{
		{0x12, 0xC6, 0xF5, 0xFF},
		{0x17, 0x67, 0xEE, 0xFF},
		{0x45, 0x30, 0xEC, 0xFF},
		{0xA1, 0x00, 0xEE, 0xFF},
	},
}

// Status colors, for warnings and urgent states.
var (
	Warning = color.RGBA{0xFF, 0xB4, 0x3E, 0xFF} // amber
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
