package overlay

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// debug traces every request and event on stderr, like libwayland does:
// WAYLAND_DEBUG=1 batox-message "hello"
var debug = os.Getenv("WAYLAND_DEBUG") != ""

var debugStart = time.Now()

func debugf(format string, a ...any) {
	if debug {
		fmt.Fprintf(os.Stderr, "[%8.3fms] %s\n", float64(time.Since(debugStart).Microseconds())/1000, fmt.Sprintf(format, a...))
	}
}

// Batocera's defaults: sway runs with XDG_RUNTIME_DIR=/var/run.
const defaultRuntimeDir = "/var/run"

type event struct {
	obj  uint32
	op   uint16
	data []byte
}

type global struct {
	name    uint32
	iface   string
	version uint32
}

// handler receives the events of one object.
type handler func(op uint16, data []byte) error

// client is one connection to the compositor.
type client struct {
	conn     *net.UnixConn
	nextID   uint32
	events   chan event
	readErr  chan error
	handlers map[uint32]handler
	names    map[uint32]string // interface of each object, for the debug trace

	registry uint32
	globals  []global
}

// procRoot is where process information is read; tests point it elsewhere.
var procRoot = "/proc"

// SocketPath returns the compositor socket the overlay connects to:
//
//  1. $XDG_RUNTIME_DIR/$WAYLAND_DISPLAY, when WAYLAND_DISPLAY is set;
//  2. otherwise (an SSH session or a system service), the socket used by the
//     programs sway started, EmulationStation first: other wayland-* sockets
//     may belong to something else, or be left over from a crash;
//  3. as a last resort, the first wayland-* socket in XDG_RUNTIME_DIR
//     (/var/run on Batocera).
func SocketPath() (string, error) {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = defaultRuntimeDir
	}
	if name := os.Getenv("WAYLAND_DISPLAY"); name != "" {
		return joinSocket(dir, name), nil
	}
	if path, ok := socketFromProcesses(); ok {
		return path, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "wayland-") && !strings.HasSuffix(e.Name(), ".lock") {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	return "", fmt.Errorf("no Wayland socket in %s: is sway running?", dir)
}

func joinSocket(dir, name string) string {
	if filepath.IsAbs(name) {
		return name
	}
	return filepath.Join(dir, name)
}

// socketFromProcesses reads WAYLAND_DISPLAY from the environment of running
// programs, preferring EmulationStation, and returns the first socket that
// exists.
func socketFromProcesses() (string, bool) {
	entries, err := os.ReadDir(procRoot)
	if err != nil {
		return "", false
	}
	self := fmt.Sprint(os.Getpid())
	var preferred, others []string
	for _, e := range entries {
		pid := e.Name()
		if pid == self || strings.Trim(pid, "0123456789") != "" {
			continue
		}
		comm, _ := os.ReadFile(filepath.Join(procRoot, pid, "comm"))
		if strings.TrimSpace(string(comm)) == "emulationstation" {
			preferred = append(preferred, pid)
		} else {
			others = append(others, pid)
		}
	}
	for _, pid := range append(preferred, others...) {
		env, err := os.ReadFile(filepath.Join(procRoot, pid, "environ"))
		if err != nil {
			continue
		}
		var display, dir string
		for _, kv := range strings.Split(string(env), "\x00") {
			if v, ok := strings.CutPrefix(kv, "WAYLAND_DISPLAY="); ok {
				display = v
			} else if v, ok := strings.CutPrefix(kv, "XDG_RUNTIME_DIR="); ok {
				dir = v
			}
		}
		if display == "" {
			continue
		}
		if dir == "" {
			dir = defaultRuntimeDir
		}
		path := joinSocket(dir, display)
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}
	return "", false
}

func dial() (*client, error) {
	path, err := SocketPath()
	if err != nil {
		return nil, err
	}
	debugf("connecting to %s", path)
	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		return nil, fmt.Errorf("connecting to the compositor at %s: %w", path, err)
	}
	c := &client{
		conn:     conn,
		nextID:   2, // 1 is wl_display
		events:   make(chan event, 128),
		readErr:  make(chan error, 1),
		handlers: map[uint32]handler{},
		names:    map[uint32]string{1: "wl_display"},
	}
	go c.readLoop()
	return c, nil
}

func (c *client) close() { _ = c.conn.Close() }

func (c *client) newID() uint32 {
	id := c.nextID
	c.nextID++
	return id
}

// newObject allocates an id and remembers its interface for the debug trace.
func (c *client) newObject(iface string) uint32 {
	id := c.newID()
	c.names[id] = iface
	return id
}

func (c *client) label(id uint32) string {
	if n, ok := c.names[id]; ok {
		return fmt.Sprintf("%s@%d", n, id)
	}
	return fmt.Sprintf("object@%d", id)
}

// words renders arguments as 32-bit words for the debug trace.
func words(b []byte) string {
	var parts []string
	for off := 0; off+4 <= len(b); off += 4 {
		v, _ := readU32(b, off)
		parts = append(parts, fmt.Sprint(v))
	}
	return strings.Join(parts, " ")
}

// send writes one request, optionally passing a file descriptor (-1 for none).
func (c *client) send(obj uint32, op uint16, a *args, fd int) error {
	var oob []byte
	if fd >= 0 {
		oob = unix.UnixRights(fd)
	}
	msg := encodeMessage(obj, op, a)
	if debug {
		fdNote := ""
		if fd >= 0 {
			fdNote = fmt.Sprintf(" +fd %d", fd)
		}
		debugf(" -> %s.request %d [%s]%s", c.label(obj), op, words(msg[8:]), fdNote)
	}
	_, _, err := c.conn.WriteMsgUnix(msg, oob, nil)
	if err != nil {
		debugf(" !! write failed: %v", err)
	}
	return err
}

func (c *client) readLoop() {
	header := make([]byte, 8)
	for {
		if _, err := io.ReadFull(c.conn, header); err != nil {
			c.readErr <- err
			return
		}
		obj, _ := readU32(header, 0)
		word, _ := readU32(header, 4)
		size := int(word >> 16)
		if size < 8 {
			c.readErr <- fmt.Errorf("invalid message size %d", size)
			return
		}
		data := make([]byte, size-8)
		if _, err := io.ReadFull(c.conn, data); err != nil {
			c.readErr <- err
			return
		}
		c.events <- event{obj: obj, op: uint16(word & 0xffff), data: data}
	}
}

// dispatch routes one event to its object's handler. Errors reported by the
// compositor (wl_display.error) are fatal.
func (c *client) dispatch(ev event) error {
	if debug {
		debugf("<-  %s.event %d [%s]", c.label(ev.obj), ev.op, words(ev.data))
	}
	if ev.obj == 1 && ev.op == displayErrorEvent {
		objID, off := readU32(ev.data, 0)
		code, off := readU32(ev.data, off)
		msg, _ := readString(ev.data, off)
		return fmt.Errorf("compositor error on object %d (code %d): %s", objID, code, msg)
	}
	if h := c.handlers[ev.obj]; h != nil {
		return h(ev.op, ev.data)
	}
	return nil
}

var errConnectionLost = errors.New("connection to the compositor lost")

// pump processes events until done returns true.
func (c *client) pump(done func() bool) error {
	for !done() {
		select {
		case ev := <-c.events:
			if err := c.dispatch(ev); err != nil {
				return err
			}
		case err := <-c.readErr:
			return fmt.Errorf("%w: %v", errConnectionLost, err)
		}
	}
	return nil
}

// roundtrip waits until the compositor has handled every earlier request.
func (c *client) roundtrip() error {
	cb := c.newObject("wl_callback")
	finished := false
	c.handlers[cb] = func(op uint16, _ []byte) error {
		if op == callbackDoneEvent {
			finished = true
		}
		return nil
	}
	defer delete(c.handlers, cb)
	if err := c.send(1, displaySync, new(args).u32(cb), -1); err != nil {
		return err
	}
	return c.pump(func() bool { return finished })
}

// loadGlobals asks for the registry and waits for the list of globals.
func (c *client) loadGlobals() error {
	c.registry = c.newObject("wl_registry")
	c.handlers[c.registry] = func(op uint16, data []byte) error {
		if op == registryGlobalEvent {
			name, off := readU32(data, 0)
			iface, off := readString(data, off)
			version, _ := readU32(data, off)
			c.globals = append(c.globals, global{name: name, iface: iface, version: version})
			debugf("     global %d: %s v%d", name, iface, version)
		}
		return nil
	}
	if err := c.send(1, displayGetRegistry, new(args).u32(c.registry), -1); err != nil {
		return err
	}
	return c.roundtrip()
}

// bind binds a global by interface name, at most at maxVersion, and returns
// the new object id and the version in use.
func (c *client) bind(iface string, maxVersion uint32) (uint32, uint32, error) {
	for _, g := range c.globals {
		if g.iface != iface {
			continue
		}
		version := min(g.version, maxVersion)
		id := c.newObject(iface)
		a := new(args).u32(g.name).str(iface).u32(version).u32(id)
		return id, version, c.send(c.registry, registryBind, a, -1)
	}
	return 0, 0, fmt.Errorf("the compositor doesn't offer %s", iface)
}
