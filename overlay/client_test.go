package overlay

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

// fakeProc points procRoot at a temporary directory, so tests never read the
// real processes (a desktop session on the developer's machine has its own
// Wayland socket).
func fakeProc(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	old := procRoot
	procRoot = root
	t.Cleanup(func() { procRoot = old })
	return root
}

// addProcess creates /proc/<pid>/comm and /proc/<pid>/environ.
func addProcess(t *testing.T, root, pid, comm string, env ...string) {
	t.Helper()
	dir := filepath.Join(root, pid)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	var environ []byte
	for _, kv := range env {
		environ = append(environ, kv...)
		environ = append(environ, 0)
	}
	if err := os.WriteFile(filepath.Join(dir, "comm"), []byte(comm+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "environ"), environ, 0o644); err != nil {
		t.Fatal(err)
	}
}

// listen creates a real Unix socket, like a compositor's.
func listen(t *testing.T, path string) {
	t.Helper()
	l, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
}

func TestSocketPathFromEnvironment(t *testing.T) {
	fakeProc(t)
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)

	t.Setenv("WAYLAND_DISPLAY", "wayland-1")
	if got, _ := SocketPath(); got != filepath.Join(dir, "wayland-1") {
		t.Errorf("relative WAYLAND_DISPLAY: got %q", got)
	}
	t.Setenv("WAYLAND_DISPLAY", "/tmp/custom")
	if got, _ := SocketPath(); got != "/tmp/custom" {
		t.Errorf("absolute WAYLAND_DISPLAY: got %q", got)
	}
}

func TestSocketPathFromEmulationStation(t *testing.T) {
	proc := fakeProc(t)
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)
	t.Setenv("WAYLAND_DISPLAY", "")

	// wayland-0 belongs to something else; sway is on wayland-1.
	listen(t, filepath.Join(dir, "wayland-0"))
	listen(t, filepath.Join(dir, "wayland-1"))
	addProcess(t, proc, "50", "other", "WAYLAND_DISPLAY=wayland-0", "XDG_RUNTIME_DIR="+dir)
	addProcess(t, proc, "900", "emulationstation", "HOME=/userdata/system", "WAYLAND_DISPLAY=wayland-1", "XDG_RUNTIME_DIR="+dir)

	got, err := SocketPath()
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(dir, "wayland-1") {
		t.Errorf("got %q, want EmulationStation's socket (wayland-1)", got)
	}
}

func TestSocketPathSkipsMissingSockets(t *testing.T) {
	proc := fakeProc(t)
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)
	t.Setenv("WAYLAND_DISPLAY", "")

	listen(t, filepath.Join(dir, "wayland-2"))
	addProcess(t, proc, "10", "emulationstation", "WAYLAND_DISPLAY=wayland-9", "XDG_RUNTIME_DIR="+dir) // gone
	addProcess(t, proc, "20", "retroarch", "WAYLAND_DISPLAY=wayland-2", "XDG_RUNTIME_DIR="+dir)

	if got, _ := SocketPath(); got != filepath.Join(dir, "wayland-2") {
		t.Errorf("got %q, want the next process's socket (wayland-2)", got)
	}
}

func TestSocketPathScanFallback(t *testing.T) {
	fakeProc(t) // no processes with WAYLAND_DISPLAY
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)
	t.Setenv("WAYLAND_DISPLAY", "")

	if _, err := SocketPath(); err == nil {
		t.Error("expected an error with no socket in the directory")
	}
	for _, name := range []string{"wayland-1.lock", "wayland-1"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if got, _ := SocketPath(); got != filepath.Join(dir, "wayland-1") {
		t.Errorf("scan: got %q, want the socket, not the lock file", got)
	}
}
