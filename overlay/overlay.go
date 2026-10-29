package overlay

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// Position is where the overlay sits on the screen. It always spans the full
// screen width; the Painter decides what is opaque.
type Position int

const (
	Top Position = iota
	Center
	Bottom
)

// ParsePosition accepts "top", "center" and "bottom".
func ParsePosition(s string) (Position, error) {
	switch s {
	case "top":
		return Top, nil
	case "center":
		return Center, nil
	case "bottom":
		return Bottom, nil
	}
	return Top, fmt.Errorf("unknown position %q: use top, center or bottom", s)
}

// Layer is the layer-shell layer of the overlay.
type Layer int

const (
	// AboveAll is drawn above everything, fullscreen games included.
	AboveAll Layer = iota
	// AboveWindows is drawn above normal windows, but below fullscreen ones.
	AboveWindows
)

// Options control where and how the overlay is shown.
type Options struct {
	Position Position
	Layer    Layer
	// Namespace identifies the overlay to the compositor (default "batox-apps").
	Namespace string
}

// Painter draws the overlay for a screen of the given width, in pixels. The
// returned image should be that wide (narrower images are centered); its
// height becomes the overlay height. Use transparent pixels for the parts of
// the screen the overlay should not cover. The image is premultiplied RGBA,
// as image.RGBA always is.
type Painter func(screenWidth int) (*image.RGBA, error)

// maxConfigureWait is how many configures paint waits for the size it asked for.
const maxConfigureWait = 5

// ErrClosed is returned when the compositor closes the overlay, for example
// because the screen it was on went away.
var ErrClosed = errors.New("the compositor closed the overlay")

// Show displays what paint draws until ctx ends. Use context.WithTimeout for
// a message that disappears on its own.
func Show(ctx context.Context, opts Options, paint Painter) error {
	return Run(ctx, opts, 0, paint)
}

// Run displays what paint draws and repaints it every interval (never when
// interval is 0) until ctx ends. Use it for content that changes, such as a
// clock or a countdown. The end of ctx is a normal stop, not an error.
func Run(ctx context.Context, opts Options, interval time.Duration, paint Painter) error {
	c, err := dial()
	if err != nil {
		return err
	}
	defer c.close()

	s, err := newSurface(c, opts)
	if err != nil {
		return err
	}
	defer s.destroy()

	if err := s.paint(paint); err != nil {
		return err
	}

	var tick <-chan time.Time
	if interval > 0 {
		t := time.NewTicker(interval)
		defer t.Stop()
		tick = t.C
	}
	for {
		select {
		case <-ctx.Done():
			debugf("     done: %v", ctx.Err())
			return nil
		case <-tick:
			if err := s.paint(paint); err != nil {
				return err
			}
		case ev := <-c.events:
			if err := c.dispatch(ev); err != nil {
				return err
			}
			if s.closed {
				return ErrClosed
			}
			if s.configured {
				// The compositor changed the size, e.g. a new screen mode.
				if err := s.ack(); err != nil {
					return err
				}
				if s.width != s.paintedWidth || s.height != s.paintedHeight {
					if err := s.paint(paint); err != nil {
						return err
					}
				}
			}
		case err := <-c.readErr:
			return fmt.Errorf("%w: %v", errConnectionLost, err)
		}
	}
}

// surface is one layer-shell surface and the buffers it has shown.
type surface struct {
	c *client

	compositorVersion   uint32
	compositor, shm     uint32
	wlSurface, layerObj uint32

	configured    bool // a configure arrived and was not acknowledged yet
	serial        uint32
	width, height uint32 // as last configured
	closed        bool

	paintedWidth, paintedHeight uint32 // size of the frame on screen

	buffers map[uint32]*shmBuffer
}

type shmBuffer struct{ mem []byte }

func newSurface(c *client, opts Options) (*surface, error) {
	if err := c.loadGlobals(); err != nil {
		return nil, err
	}
	s := &surface{c: c, buffers: map[uint32]*shmBuffer{}}
	var err error
	if s.compositor, s.compositorVersion, err = c.bind("wl_compositor", 4); err != nil {
		return nil, err
	}
	if s.shm, _, err = c.bind("wl_shm", 1); err != nil {
		return nil, err
	}
	layerShell, _, err := c.bind("zwlr_layer_shell_v1", 1)
	if err != nil {
		return nil, fmt.Errorf("%w; this compositor can't show overlays", err)
	}

	// A surface that ignores all input: an empty input region, so taps and
	// clicks go through, and no keyboard interactivity (set below).
	s.wlSurface = c.newObject("wl_surface")
	if err := c.send(s.compositor, compositorCreateSurface, new(args).u32(s.wlSurface), -1); err != nil {
		return nil, err
	}
	region := c.newObject("wl_region")
	if err := c.send(s.compositor, compositorCreateRegion, new(args).u32(region), -1); err != nil {
		return nil, err
	}
	if err := c.send(s.wlSurface, surfaceSetInputRegion, new(args).u32(region), -1); err != nil {
		return nil, err
	}
	if err := c.send(region, regionDestroy, nil, -1); err != nil {
		return nil, err
	}

	layer := uint32(layerOverlay)
	if opts.Layer == AboveWindows {
		layer = layerTop
	}
	namespace := opts.Namespace
	if namespace == "" {
		namespace = "batox-apps"
	}
	s.layerObj = c.newObject("zwlr_layer_surface_v1")
	c.handlers[s.layerObj] = s.onLayerEvent
	a := new(args).u32(s.layerObj).u32(s.wlSurface).u32(0 /* any output */).u32(layer).str(namespace)
	if err := c.send(layerShell, layerShellGetLayerSurface, a, -1); err != nil {
		return nil, err
	}

	anchor := uint32(anchorLeft | anchorRight)
	switch opts.Position {
	case Top:
		anchor |= anchorTop
	case Bottom:
		anchor |= anchorBottom
	}
	requests := []struct {
		op uint16
		a  *args
	}{
		{layerSurfaceSetAnchor, new(args).u32(anchor)},
		{layerSurfaceSetSize, new(args).u32(0).u32(1)}, // width 0: the full width; the real height comes with the first paint
		{layerSurfaceSetKeyboardInteractivity, new(args).u32(0)},
	}
	for _, r := range requests {
		if err := c.send(s.layerObj, r.op, r.a, -1); err != nil {
			return nil, err
		}
	}
	// The first commit carries no buffer: it asks the compositor for a size.
	if err := c.send(s.wlSurface, surfaceCommit, nil, -1); err != nil {
		return nil, err
	}
	if err := s.awaitConfigure(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *surface) onLayerEvent(op uint16, data []byte) error {
	switch op {
	case layerSurfaceConfigureEvent:
		serial, off := readU32(data, 0)
		width, off := readU32(data, off)
		height, _ := readU32(data, off)
		s.configured, s.serial, s.width, s.height = true, serial, width, height
		debugf("     configure: serial %d, %dx%d", serial, width, height)
	case layerSurfaceClosedEvent:
		debugf("     closed by the compositor")
		s.closed = true
	}
	return nil
}

// ack acknowledges the last configure.
func (s *surface) ack() error {
	s.configured = false
	return s.c.send(s.layerObj, layerSurfaceAckConfigure, new(args).u32(s.serial), -1)
}

// awaitConfigure waits for the compositor to configure the surface, then
// acknowledges it.
func (s *surface) awaitConfigure() error {
	if err := s.c.pump(func() bool { return s.configured || s.closed }); err != nil {
		return err
	}
	if s.closed {
		return ErrClosed
	}
	return s.ack()
}

// paint draws a new frame, resizing the surface first when its height changes.
func (s *surface) paint(paint Painter) error {
	if s.width == 0 {
		return errors.New("the compositor gave the overlay no width")
	}
	img, err := paint(int(s.width))
	if err != nil {
		return err
	}
	h := img.Bounds().Dy()
	debugf("     painter returned %dx%d for a %d px wide screen", img.Bounds().Dx(), h, s.width)
	if debug {
		dumpFrame(img)
	}
	if h <= 0 {
		return errors.New("the painter returned an empty image")
	}
	if uint32(h) != s.height {
		if err := s.c.send(s.layerObj, layerSurfaceSetSize, new(args).u32(0).u32(uint32(h)), -1); err != nil {
			return err
		}
		if err := s.c.send(s.wlSurface, surfaceCommit, nil, -1); err != nil {
			return err
		}
		// sway may first answer with configures for earlier commits, still
		// carrying the old height: wait for the one with the height asked for.
		// The limit covers a compositor that never gives that exact height.
		for i := 0; i < maxConfigureWait; i++ {
			if err := s.awaitConfigure(); err != nil {
				return err
			}
			if s.height == uint32(h) {
				break
			}
			debugf("     still %dx%d, waiting for the height %d", s.width, s.height, h)
		}
	}

	width, height := int(s.width), int(s.height)
	bufferID, buffer, err := s.newBuffer(width, height)
	if err != nil {
		return err
	}
	blitARGB8888(img, buffer.mem, width, height)

	damage := uint16(surfaceDamage)
	if s.compositorVersion >= 4 {
		damage = surfaceDamageBuffer
	}
	steps := []struct {
		op uint16
		a  *args
	}{
		{surfaceAttach, new(args).u32(bufferID).i32(0).i32(0)},
		{damage, new(args).i32(0).i32(0).i32(int32(width)).i32(int32(height))},
		{surfaceCommit, nil},
	}
	for _, st := range steps {
		if err := s.c.send(s.wlSurface, st.op, st.a, -1); err != nil {
			return err
		}
	}
	s.paintedWidth, s.paintedHeight = s.width, s.height
	return nil
}

// newBuffer creates a shared-memory buffer. The compositor reads it after the
// commit and sends wl_buffer.release when done; only then is it freed.
func (s *surface) newBuffer(width, height int) (uint32, *shmBuffer, error) {
	stride := width * 4
	size := stride * height
	fd, err := unix.MemfdCreate("batox-apps", unix.MFD_CLOEXEC)
	if err != nil {
		return 0, nil, fmt.Errorf("creating shared memory: %w", err)
	}
	defer unix.Close(fd) // the compositor keeps its own copy of the descriptor
	if err := unix.Ftruncate(fd, int64(size)); err != nil {
		return 0, nil, err
	}
	mem, err := unix.Mmap(fd, 0, size, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		return 0, nil, err
	}

	pool := s.c.newObject("wl_shm_pool")
	if err := s.c.send(s.shm, shmCreatePool, new(args).u32(pool).i32(int32(size)), fd); err != nil {
		_ = unix.Munmap(mem)
		return 0, nil, err
	}
	id := s.c.newObject("wl_buffer")
	debugf("     new %dx%d buffer, %d bytes", width, height, size)
	a := new(args).u32(id).i32(0).i32(int32(width)).i32(int32(height)).i32(int32(stride)).u32(shmFormatARGB8888)
	if err := s.c.send(pool, shmPoolCreateBuffer, a, -1); err != nil {
		_ = unix.Munmap(mem)
		return 0, nil, err
	}
	if err := s.c.send(pool, shmPoolDestroy, nil, -1); err != nil {
		_ = unix.Munmap(mem)
		return 0, nil, err
	}

	buf := &shmBuffer{mem: mem}
	s.buffers[id] = buf
	s.c.handlers[id] = func(op uint16, _ []byte) error {
		if op == bufferReleaseEvent {
			s.freeBuffer(id)
		}
		return nil
	}
	return id, buf, nil
}

func (s *surface) freeBuffer(id uint32) {
	buf, ok := s.buffers[id]
	if !ok {
		return
	}
	_ = s.c.send(id, bufferDestroy, nil, -1)
	_ = unix.Munmap(buf.mem)
	delete(s.buffers, id)
	delete(s.c.handlers, id)
}

// destroy removes the overlay from the screen and frees everything.
func (s *surface) destroy() {
	_ = s.c.send(s.layerObj, layerSurfaceDestroy, nil, -1)
	_ = s.c.send(s.wlSurface, surfaceDestroy, nil, -1)
	for id := range s.buffers {
		s.freeBuffer(id)
	}
	_ = s.c.roundtrip()
}

// blitARGB8888 copies a premultiplied RGBA image into a width x height
// ARGB8888 buffer (B, G, R, A in memory), centered horizontally and clipped.
func blitARGB8888(img *image.RGBA, dst []byte, width, height int) {
	clear(dst)
	b := img.Bounds()
	offX := (width - b.Dx()) / 2
	for y := 0; y < b.Dy() && y < height; y++ {
		for x := 0; x < b.Dx(); x++ {
			dx := x + offX
			if dx < 0 || dx >= width {
				continue
			}
			si := img.PixOffset(b.Min.X+x, b.Min.Y+y)
			di := (y*width + dx) * 4
			dst[di+0] = img.Pix[si+2]
			dst[di+1] = img.Pix[si+1]
			dst[di+2] = img.Pix[si+0]
			dst[di+3] = img.Pix[si+3]
		}
	}
}

// dumpFrame saves the last painted image, to tell drawing problems from
// display problems while debugging.
func dumpFrame(img *image.RGBA) {
	const path = "/tmp/batox-apps-frame.png"
	f, err := os.Create(path)
	if err != nil {
		debugf("     could not save the frame: %v", err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		debugf("     could not save the frame: %v", err)
		return
	}
	debugf("     frame saved to %s", path)
}
