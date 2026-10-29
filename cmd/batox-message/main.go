// batox-message shows a message on top of everything on a Batocera console,
// games included, for a few seconds.
//
//	batox-message "Dinner is ready!"
//	batox-message -seconds 10 -position center -title "Mom" "Time for bed."
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/monteirohat/batox-apps/overlay"
	"github.com/monteirohat/batox-apps/ui"
)

// version is set at build time: -ldflags "-X main.version=v1.0.0".
var version = "dev"

func main() {
	seconds := flag.Float64("seconds", 6, "how long the message stays on screen")
	position := flag.String("position", "top", "where the message appears: top, center or bottom")
	title := flag.String("title", "BatoX", `small line above the message ("" for none)`)
	scale := flag.Float64("scale", 1, "text size multiplier")
	verbose := flag.Bool("verbose", false, "print the compositor socket used")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: batox-message [options] \"message\"\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}
	text := strings.TrimSpace(strings.Join(flag.Args(), " "))
	text = strings.ReplaceAll(text, `\n`, "\n") // "line one\nline two" from a shell
	if text == "" || *seconds <= 0 {
		flag.Usage()
		os.Exit(2)
	}
	pos, err := overlay.ParsePosition(*position)
	if err != nil {
		fail(err)
	}
	if *verbose {
		socket, err := overlay.SocketPath()
		if err != nil {
			fail(err)
		}
		fmt.Fprintln(os.Stderr, "batox-message: compositor socket:", socket)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*seconds*float64(time.Second)))
	defer cancel()
	card := ui.Card{Title: *title, Text: text, Scale: *scale}
	opts := overlay.Options{Position: pos, Namespace: "batox-message"}
	if err := overlay.Show(ctx, opts, card.Render); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "batox-message:", err)
	os.Exit(1)
}
