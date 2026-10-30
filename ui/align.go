package ui

// Align places a component horizontally in the overlay, which always spans
// the screen width. Together with overlay.Top or overlay.Bottom, AlignLeft and
// AlignRight put a component in a corner of the screen.
type Align int

const (
	AlignCenter Align = iota
	AlignLeft
	AlignRight
)

// x returns the left edge of a component width wide on a screen screenWidth
// wide, keeping margin from the screen edge when aligned to a side.
func (a Align) x(screenWidth, width, margin int) int {
	switch a {
	case AlignLeft:
		return margin
	case AlignRight:
		return screenWidth - width - margin
	}
	return (screenWidth - width) / 2
}
