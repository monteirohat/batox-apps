// batox-timer shows a countdown in a corner of the screen, on top of the game.
// When it reaches zero, it closes the running game or powers the console off.
//
//	batox-timer -duration 30m
//	batox-timer -duration 1h -corner bottom-left -action shutdown
//	batox-timer -duration 2m -dry-run          # try it without closing anything
//
// Stopping it early (Ctrl+C, or pkill batox-timer) removes the timer and does
// nothing else.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"image"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/monteirohat/batox-apps/overlay"
	"github.com/monteirohat/batox-apps/ui"
)

// version is set at build time: -ldflags "-X main.version=v1.0.0".
var version = "dev"

// timesUpFor is how long "Time's up" stays on screen before the action.
const timesUpFor = 4 * time.Second

const maxDuration = 24 * time.Hour

// parseCorner turns "top-right" and the like into an overlay position and a
// horizontal alignment.
func parseCorner(s string) (overlay.Position, ui.Align, error) {
	switch s {
	case "top-left":
		return overlay.Top, ui.AlignLeft, nil
	case "top-right":
		return overlay.Top, ui.AlignRight, nil
	case "bottom-left":
		return overlay.Bottom, ui.AlignLeft, nil
	case "bottom-right":
		return overlay.Bottom, ui.AlignRight, nil
	}
	return 0, 0, fmt.Errorf("unknown corner %q: use top-left, top-right, bottom-left or bottom-right", s)
}

func main() {
	duration := flag.Duration("duration", 30*time.Minute, "countdown length, such as 45m, 1h or 1h30m")
	corner := flag.String("corner", "top-right", "where the timer appears: top-left, top-right, bottom-left or bottom-right")
	actionFlag := flag.String("action", string(CloseGame), "what happens at zero: close-game, shutdown or none")
	title := flag.String("title", "Time left", `small line above the time ("" for none)`)
	doneTitle := flag.String("done-title", "Time's up", "the title once the time is over")
	scale := flag.Float64("scale", 1, "size multiplier")
	esAPI := flag.String("es-api", "http://127.0.0.1:1234", "EmulationStation's web API, used to close the game")
	dryRun := flag.Bool("dry-run", false, "at zero, only print what would happen")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: batox-timer [options]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}
	if *duration <= 0 || *duration > maxDuration {
		fail(fmt.Errorf("the duration must be between 1s and %v", maxDuration))
	}
	pos, align, err := parseCorner(*corner)
	if err != nil {
		fail(err)
	}
	action, err := parseAction(*actionFlag)
	if err != nil {
		fail(err)
	}

	// SIGINT and SIGTERM cancel the timer: the action never runs then.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	end := time.Now().Add(*duration)
	timer := ui.Countdown{Title: *title, Total: *duration, Align: align, Scale: *scale}
	opts := overlay.Options{Position: pos, Namespace: "batox-timer"}

	countdown, cancel := context.WithDeadline(ctx, end)
	err = overlay.Run(countdown, opts, time.Second, func(screenWidth int) (*image.RGBA, error) {
		t := timer
		t.Remaining = time.Until(end)
		return t.Render(screenWidth)
	})
	cancel()
	if err != nil {
		fail(err)
	}
	if cancelled(ctx) {
		return
	}

	// Time's up: say so for a moment, then act.
	final := timer
	final.Title, final.Remaining = *doneTitle, 0
	shown, cancel := context.WithTimeout(ctx, timesUpFor)
	err = overlay.Show(shown, opts, final.Render)
	cancel()
	if err != nil && !errors.Is(err, overlay.ErrClosed) {
		fmt.Fprintln(os.Stderr, "batox-timer:", err)
	}
	if cancelled(ctx) {
		return
	}

	if *dryRun {
		fmt.Printf("batox-timer: time's up; would run %q (dry run, nothing done)\n", action)
		return
	}
	actCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := newActor(*esAPI).run(actCtx, action)
	if err != nil {
		fail(err)
	}
	fmt.Println("batox-timer: time's up;", result)
}

// cancelled reports, and says, whether the timer was stopped early.
func cancelled(ctx context.Context) bool {
	if ctx.Err() == nil {
		return false
	}
	fmt.Fprintln(os.Stderr, "batox-timer: stopped before the end; nothing was done")
	return true
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "batox-timer:", strings.TrimSpace(err.Error()))
	os.Exit(1)
}
