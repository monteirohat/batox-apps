package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/monteirohat/batox-apps/overlay"
	"github.com/monteirohat/batox-apps/ui"
)

func TestParseCorner(t *testing.T) {
	cases := map[string]struct {
		pos   overlay.Position
		align ui.Align
	}{
		"top-left":     {overlay.Top, ui.AlignLeft},
		"top-right":    {overlay.Top, ui.AlignRight},
		"bottom-left":  {overlay.Bottom, ui.AlignLeft},
		"bottom-right": {overlay.Bottom, ui.AlignRight},
	}
	for s, want := range cases {
		pos, align, err := parseCorner(s)
		if err != nil || pos != want.pos || align != want.align {
			t.Errorf("parseCorner(%q) = %v, %v, %v", s, pos, align, err)
		}
	}
	if _, _, err := parseCorner("middle"); err == nil {
		t.Error("expected an error for an unknown corner")
	}
}

func TestParseAction(t *testing.T) {
	for _, s := range []string{"close-game", "shutdown", "none"} {
		if a, err := parseAction(s); err != nil || string(a) != s {
			t.Errorf("parseAction(%q) = %q, %v", s, a, err)
		}
	}
	if _, err := parseAction("reboot"); err == nil {
		t.Error("expected an error for an unknown action")
	}
}

// fakeES answers like EmulationStation's web API and counts /emukill calls.
type fakeES struct {
	mu      sync.Mutex
	playing bool
	kills   int
}

func (f *fakeES) server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		switch r.URL.Path {
		case "/runningGame":
			if !f.playing {
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"msg":"NO GAME RUNNING"}`))
				return
			}
			_, _ = w.Write([]byte(`{"name":"Golden Sun"}`))
		case "/emukill":
			f.kills++
			f.playing = false
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func (f *fakeES) killCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.kills
}

func TestCloseGame(t *testing.T) {
	es := &fakeES{playing: true}
	a := newActor(es.server(t).URL)
	result, err := a.run(context.Background(), CloseGame)
	if err != nil || result != "the game was closed" || es.killCount() != 1 {
		t.Errorf("run = %q, %v; kills %d", result, err, es.killCount())
	}
}

func TestCloseGameWithNoGame(t *testing.T) {
	es := &fakeES{playing: false}
	a := newActor(es.server(t).URL)
	result, err := a.run(context.Background(), CloseGame)
	if err != nil || result != "no game was running" || es.killCount() != 0 {
		t.Errorf("run = %q, %v; kills %d", result, err, es.killCount())
	}
}

func TestCloseGameWithoutEmulationStation(t *testing.T) {
	a := newActor("http://127.0.0.1:1") // nothing listens on port 1
	if _, err := a.run(context.Background(), CloseGame); err == nil {
		t.Error("expected an error when EmulationStation is not reachable")
	}
}

func TestShutdown(t *testing.T) {
	a := newActor("http://127.0.0.1:1")
	called := false
	a.poweroff = func(context.Context) error { called = true; return nil }
	if _, err := a.run(context.Background(), Shutdown); err != nil || !called {
		t.Errorf("shutdown: err %v, poweroff called %v", err, called)
	}

	a.poweroff = func(context.Context) error { return errors.New("not permitted") }
	if _, err := a.run(context.Background(), Shutdown); err == nil {
		t.Error("expected the poweroff error")
	}
}

func TestNothing(t *testing.T) {
	a := newActor("http://127.0.0.1:1")
	a.poweroff = func(context.Context) error { t.Fatal("poweroff must not run"); return nil }
	if _, err := a.run(context.Background(), Nothing); err != nil {
		t.Error(err)
	}
}
