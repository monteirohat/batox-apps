// Package overlay shows images on top of everything on a Batocera console,
// fullscreen games included.
//
// Batocera runs sway, a Wayland compositor. This package talks to it directly
// (pure Go, no CGO, no system libraries) and places a surface on the
// "overlay" layer of the wlr-layer-shell protocol, which sway draws above
// fullscreen windows such as RetroArch. The surface takes no keyboard,
// controller or pointer input, so the game keeps the controls.
//
// An app only provides a Painter: a function that draws the overlay for a
// given screen width. Show displays it once; Run also repaints it on a timer,
// for content that changes, such as a clock or a countdown.
//
//	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
//	defer cancel()
//	err := overlay.Show(ctx, overlay.Options{Position: overlay.Top}, card.Render)
package overlay
