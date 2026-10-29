# Creating an app

An app is a folder in `cmd/` with a `main.go`. The Makefile and the release workflow find it
automatically, so nothing else has to change.

## 1. Create the folder

```bash
mkdir cmd/batox-countdown
```

Name apps `batox-<what-it-does>`; the folder name becomes the binary name.

## 2. Write main.go

This example shows a countdown that repaints every second, using `overlay.Run`:

```go
// batox-countdown shows a countdown on top of the game.
//
//	batox-countdown -minutes 5 "Time to stop playing"
package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"os"
	"time"

	"github.com/monteirohat/batox-apps/overlay"
	"github.com/monteirohat/batox-apps/ui"
)

func main() {
	minutes := flag.Float64("minutes", 5, "countdown length")
	flag.Parse()
	title := flag.Arg(0)

	end := time.Now().Add(time.Duration(*minutes * float64(time.Minute)))
	ctx, cancel := context.WithDeadline(context.Background(), end)
	defer cancel()

	paint := func(screenWidth int) (*image.RGBA, error) {
		left := time.Until(end).Round(time.Second)
		card := ui.Card{Title: title, Text: fmt.Sprintf("%d:%02d", int(left.Minutes()), int(left.Seconds())%60)}
		return card.Render(screenWidth)
	}
	opts := overlay.Options{Position: overlay.Top, Namespace: "batox-countdown"}
	if err := overlay.Run(ctx, opts, time.Second, paint); err != nil {
		fmt.Fprintln(os.Stderr, "batox-countdown:", err)
		os.Exit(1)
	}
}
```

## 3. Choose Show or Run

| Function | Use it when |
|---|---|
| `overlay.Show(ctx, opts, paint)` | the content doesn't change: a message, a notice |
| `overlay.Run(ctx, opts, interval, paint)` | the content changes: a clock, a countdown, progress |

Both stop when the context ends: use `context.WithTimeout` or `WithDeadline` for overlays that
disappear on their own.

## 4. Draw

A Painter receives the screen width and returns an image as wide as the screen; its height becomes
the overlay height. Keep everything outside your content transparent.

- Reuse `ui.Card` when a card fits.
- For something new, build it from `ui.Face`, `ui.Wrap`, `ui.DrawText`, `ui.FillRoundedRect` and
  `ui.Gradient`, and derive every size from the screen width, as `Card` does, so it looks the same
  at 1080p and 4K.
- A component that more than one app could use belongs in `ui/`, with tests.

## 5. Try it on the console

```bash
make deploy HOST=batocera.local        # builds for the Pi and copies every app
ssh root@batocera.local
/userdata/system/batox-apps/batox-countdown -minutes 1 "Test"
```

Open a game first to check that the overlay appears on top of it and that the controller keeps
working.

## 6. Before committing

```bash
make fmt && make vet && make test
```

Add the app to the list in the README and a line to the `Unreleased` section of `CHANGELOG.md`.
