package overlay

// The Wayland wire protocol: every message starts with the object id (uint32)
// and a word holding the message size in the high 16 bits and the opcode in
// the low 16. Arguments are 32-bit words; strings are length-prefixed,
// NUL-terminated and padded to 32 bits. File descriptors travel separately,
// as SCM_RIGHTS ancillary data.

import "encoding/binary"

// Requests and events used by this package, by interface.
const (
	displaySync        = 0
	displayGetRegistry = 1
	displayErrorEvent  = 0

	registryBind        = 0
	registryGlobalEvent = 0

	callbackDoneEvent = 0

	compositorCreateSurface = 0
	compositorCreateRegion  = 1

	shmCreatePool = 0

	shmPoolCreateBuffer = 0
	shmPoolDestroy      = 1

	bufferDestroy      = 0
	bufferReleaseEvent = 0

	surfaceDestroy        = 0
	surfaceAttach         = 1
	surfaceDamage         = 2
	surfaceSetInputRegion = 5
	surfaceCommit         = 6
	surfaceDamageBuffer   = 9 // wl_surface v4

	regionDestroy = 0

	layerShellGetLayerSurface = 0

	layerSurfaceSetSize                  = 0
	layerSurfaceSetAnchor                = 1
	layerSurfaceSetKeyboardInteractivity = 4
	layerSurfaceAckConfigure             = 6
	layerSurfaceDestroy                  = 7
	layerSurfaceConfigureEvent           = 0
	layerSurfaceClosedEvent              = 1
)

// zwlr_layer_shell_v1 and wl_shm values.
const (
	layerTop     = 2
	layerOverlay = 3

	anchorTop    = 1
	anchorBottom = 2
	anchorLeft   = 4
	anchorRight  = 8

	shmFormatARGB8888 = 0
)

// args builds the argument part of a request.
type args struct{ b []byte }

func (a *args) u32(v uint32) *args {
	a.b = binary.LittleEndian.AppendUint32(a.b, v)
	return a
}

func (a *args) i32(v int32) *args { return a.u32(uint32(v)) }

func (a *args) str(s string) *args {
	n := len(s) + 1 // with the terminating NUL
	a.u32(uint32(n))
	a.b = append(a.b, s...)
	a.b = append(a.b, 0)
	for len(a.b)%4 != 0 {
		a.b = append(a.b, 0)
	}
	return a
}

// encodeMessage returns a complete request: header followed by arguments.
func encodeMessage(obj uint32, op uint16, a *args) []byte {
	var payload []byte
	if a != nil {
		payload = a.b
	}
	size := 8 + len(payload)
	msg := make([]byte, 0, size)
	msg = binary.LittleEndian.AppendUint32(msg, obj)
	msg = binary.LittleEndian.AppendUint32(msg, uint32(size)<<16|uint32(op))
	return append(msg, payload...)
}

// readU32 reads a word at off and returns the offset after it.
func readU32(data []byte, off int) (uint32, int) {
	if off+4 > len(data) {
		return 0, len(data)
	}
	return binary.LittleEndian.Uint32(data[off:]), off + 4
}

// readString reads a string argument at off and returns the offset after it.
func readString(data []byte, off int) (string, int) {
	n, off := readU32(data, off)
	if n == 0 || off+int(n) > len(data) {
		return "", off
	}
	s := string(data[off : off+int(n)-1])
	return s, off + (int(n)+3)&^3
}
