package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// Action is what happens when the countdown reaches zero.
type Action string

const (
	CloseGame Action = "close-game" // close the running game, if any
	Shutdown  Action = "shutdown"   // power the console off
	Nothing   Action = "none"       // only show the timer
)

func parseAction(s string) (Action, error) {
	switch a := Action(s); a {
	case CloseGame, Shutdown, Nothing:
		return a, nil
	}
	return "", fmt.Errorf("unknown action %q: use close-game, shutdown or none", s)
}

// actor carries out actions. Its fields point at the real system and are
// replaced in tests.
type actor struct {
	esAPI    string                          // EmulationStation's web API
	poweroff func(ctx context.Context) error // powers the console off
	client   *http.Client
}

func newActor(esAPI string) *actor {
	return &actor{
		esAPI: strings.TrimRight(esAPI, "/"),
		poweroff: func(ctx context.Context) error {
			// The same command BatoX uses for Shut down.
			return exec.CommandContext(ctx, "poweroff").Run()
		},
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// run carries out the action and describes what it did.
func (a *actor) run(ctx context.Context, action Action) (string, error) {
	switch action {
	case CloseGame:
		return a.closeGame(ctx)
	case Shutdown:
		if err := a.poweroff(ctx); err != nil {
			return "", fmt.Errorf("powering off: %w", err)
		}
		return "the console is shutting down", nil
	}
	return "nothing to do", nil
}

// closeGame closes the running game through EmulationStation's web API:
// /runningGame answers 201 when no game is running, /emukill closes it.
func (a *actor) closeGame(ctx context.Context) (string, error) {
	status, body, err := a.get(ctx, "/runningGame")
	if err != nil {
		return "", err
	}
	if status == http.StatusCreated {
		return "no game was running", nil
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("EmulationStation /runningGame answered HTTP %d: %s", status, strings.TrimSpace(body))
	}
	if status, body, err = a.get(ctx, "/emukill"); err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("EmulationStation /emukill answered HTTP %d: %s", status, strings.TrimSpace(body))
	}
	return "the game was closed", nil
}

func (a *actor) get(ctx context.Context, route string) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.esAPI+route, nil)
	if err != nil {
		return 0, "", err
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("EmulationStation is not reachable at %s: %w", a.esAPI, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return resp.StatusCode, string(body), nil
}
