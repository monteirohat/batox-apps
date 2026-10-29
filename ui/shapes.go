package ui

import (
	"image"
	"image/color"
	"math"
)

// FillRoundedRect fills r with rounded corners and anti-aliased edges. colorAt
// gives the color of each pixel, relative to r (0,0 is the top-left corner),
// which allows gradients and accent bars. Pixels are blended over what is
// already in img.
func FillRoundedRect(img *image.RGBA, r image.Rectangle, radius int, colorAt func(x, y int) color.RGBA) {
	w, h := float64(r.Dx()), float64(r.Dy())
	rad := math.Min(float64(radius), math.Min(w, h)/2)
	for y := 0; y < r.Dy(); y++ {
		for x := 0; x < r.Dx(); x++ {
			cover := roundedCover(float64(x)+0.5, float64(y)+0.5, w, h, rad)
			if cover <= 0 {
				continue
			}
			px, py := r.Min.X+x, r.Min.Y+y
			if !(image.Point{px, py}.In(img.Bounds())) {
				continue
			}
			blend(img, px, py, colorAt(x, y), cover)
		}
	}
}

// roundedCover is how much of the pixel centered at (px, py) lies inside a
// w x h rounded rectangle, from 0 to 1.
func roundedCover(px, py, w, h, r float64) float64 {
	dx := math.Max(math.Max(r-px, px-(w-r)), 0)
	dy := math.Max(math.Max(r-py, py-(h-r)), 0)
	return clamp01(0.5 - (math.Hypot(dx, dy) - r))
}

// blend draws c with extra coverage over the pixel at (x, y), premultiplied.
func blend(img *image.RGBA, x, y int, c color.RGBA, cover float64) {
	a := float64(c.A) / 255 * cover
	i := img.PixOffset(x, y)
	p := img.Pix[i : i+4 : i+4]
	inv := 1 - a
	p[0] = uint8(float64(c.R)*a + float64(p[0])*inv + 0.5)
	p[1] = uint8(float64(c.G)*a + float64(p[1])*inv + 0.5)
	p[2] = uint8(float64(c.B)*a + float64(p[2])*inv + 0.5)
	p[3] = uint8(255*a + float64(p[3])*inv + 0.5)
}
