# Architecture

BatoX Apps are small programs that draw on top of everything on a Batocera console, games
included. They share two libraries, so an app is usually a single short `main.go`.

```
cmd/<app>/     the apps: parse flags, build what to show, call overlay
ui/            what to draw: theme, fonts, text wrapping, shapes, components (Card)
overlay/       how to show it: the connection to sway and the on-screen surface
```

An app depends on `ui` and `overlay`; `ui` and `overlay` don't depend on each other. They meet in
one function type:

```go
type Painter func(screenWidth int) (*image.RGBA, error)
```

`overlay` asks for an image as wide as the screen; `ui` components (or any code) draw one.

## How the overlay works

Batocera 43 runs **sway**, a Wayland compositor from the wlroots family. EmulationStation and the
emulators are sway windows; a game runs as a fullscreen window (for example RetroArch, with the
`app_id` `com.libretro.RetroArch`).

wlroots compositors support the **wlr-layer-shell** protocol, which lets a program place a surface
on a layer instead of opening a window. The **overlay** layer is drawn above fullscreen windows,
which is what puts our messages on top of games. The **top** layer (`overlay.AboveWindows`) is
drawn above normal windows but below fullscreen ones.

The surface is created so that it never takes input:

- `keyboard_interactivity` is `none`, so the keyboard and controllers stay with the game;
- the input region is empty, so pointer events go through it.

### No CGO, no system libraries

`overlay` speaks the Wayland wire protocol directly over the compositor's Unix socket:

- `wire.go`: message encoding (object id, size and opcode header; 32-bit arguments; padded strings)
  and the opcodes of the few interfaces used;
- `client.go`: the connection, the event reader, per-object event handlers, roundtrips, and binding
  globals (`wl_compositor`, `wl_shm`, `zwlr_layer_shell_v1`);
- `overlay.go`: the public API and the layer surface.

Pixels go through shared memory: each frame is a `memfd` mapped into the app, passed to sway with
`SCM_RIGHTS`, in the `ARGB8888` format (premultiplied, B, G, R, A in memory). A buffer is freed only
after sway sends `wl_buffer.release`, so repainting never touches memory sway is still reading.

Fonts are embedded (Go Regular and Go Bold, from `golang.org/x/image`), so the apps don't depend on
the fonts installed on the console. The result is a static binary that runs on any Batocera with
sway, on arm64 and amd64.

### Life cycle

1. Find sway's socket (`overlay.SocketPath`): `$XDG_RUNTIME_DIR/$WAYLAND_DISPLAY` when set;
   otherwise (an SSH session or a system service) the socket in the environment of the programs
   sway started, EmulationStation first; as a last resort, the first `wayland-*` socket in
   `/var/run`. The last resort alone is not enough: `/var/run` can hold other `wayland-*` sockets
   (Batocera's sway listens on `wayland-1`).
2. Read the globals and bind the three interfaces.
3. Create a surface with an empty input region and a layer surface anchored left and right (full
   width) and to the top, the bottom, or neither (center).
4. Commit without a buffer; sway answers with a `configure` carrying the screen width.
5. Call the Painter with that width; resize the surface to the image height and wait for the
   `configure` carrying that height. sway can first answer with configures for earlier commits,
   still with the old height: accepting the first configure that arrives draws a frame of the wrong
   size (an early version showed a 1-pixel-tall overlay this way).
6. Copy the image into a shared-memory buffer, attach it and commit: the overlay appears.
7. `Show` waits until its context ends; `Run` also repaints on a timer. Then the surface is destroyed
   and the overlay disappears.

### Limits

- **Input.** Overlays don't take input by design. An interactive app (a menu, an on-screen
  keyboard) would need keyboard interactivity, which takes the controller away from the game while
  it is open.
- **Performance.** A message shown for a few seconds is cheap. Something always on screen should
  repaint only when its content changes, and stay small.
- **sway.** Everything relies on Batocera using a wlroots compositor (sway, as of Batocera 43). The
  dependency is isolated in `overlay`.

## Using the libraries from another program

BatoX and other Go programs can import the libraries and show overlays without a separate binary:

```go
import (
	"github.com/monteirohat/batox-apps/overlay"
	"github.com/monteirohat/batox-apps/ui"
)
```

A program running as a Batocera service has no `WAYLAND_DISPLAY`; `overlay.SocketPath` finds
sway's socket through EmulationStation's environment anyway.
