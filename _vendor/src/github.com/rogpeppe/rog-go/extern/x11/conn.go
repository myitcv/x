// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This package implements an X11 backend for the exp/draw package.
//
// The X protocol specification is at ftp://ftp.x.org/pub/X11R7.0/doc/PDF/proto.pdf.
// A summary of the wire format can be found in XCB's xproto.xml.
package x11

// BUG(nigeltao): This is a toy library and not ready for production use.

import (
	"bufio"
	"errors"
	"exp/draw"
	"image"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type resID uint32 // X resource IDs.

// TODO(nigeltao): Handle window resizes.
const (
	windowHeight = 600
	windowWidth  = 800
)

type conn struct {
	// TODO(nigeltao): Figure out which goroutine should be responsible for closing c,
	// or if there is a race condition if one goroutine calls c.Close whilst another one
	// is reading from r, or writing to w.
	c io.Closer
	r *bufio.Reader
	w *bufio.Writer

	gc, window, root, visual resID

	img        *image.RGBA
	bufimg     *image.RGBA     // coherent image, as of last FlushImage.
	dirty      image.Rectangle // of bufimg that needs to be flushed to server.
	flushLock  sync.Mutex
	event      chan interface{}
	mouseState draw.MouseEvent

	buf [256]byte // General purpose scratch buffer.

	flush     chan bool
	flushBuf0 [24]byte
	flushBuf1 [4 * 1024]byte
}

// flusher runs in its own goroutine, serving both FlushImage calls directly from the exp/draw client
// and indirectly from X expose events. It paints c.img to the X server via PutImage requests.
func (c *conn) flusher() {
	imgr := c.img.Bounds()
	for {
		_ = <-c.flush
		if closed(c.flush) {
			return
		}
		c.flushLock.Lock()
		dirty := c.dirty
		c.dirty = image.ZR

		// sanity check against dubious flush rectangles
		if dirty = dirty.Intersect(imgr); dirty.Empty() {
			c.flushLock.Unlock()
			continue
		}

		// Each X request has a 16-bit length (in terms of 4-byte units). To avoid going over
		// this limit, we send PutImage for each row of the image, rather than trying to paint
		// the entire image in one X request. This approach could easily be optimized (or the
		// X protocol may have an escape sequence to delimit very large requests).
		// TODO(nigeltao): See what XCB's xcb_put_image does in this situation.
		w, h := dirty.Dx(), dirty.Dy()
		units := 6 + w
		if units > 0xffff || h > 0xffff {
			// This window is too large for X.
			c.flushLock.Unlock()
			return
		}

		c.flushBuf0[0] = 0x48 // PutImage opcode.
		c.flushBuf0[1] = 0x02 // XCB_IMAGE_FORMAT_Z_PIXMAP.
		c.flushBuf0[2] = uint8(units)
		c.flushBuf0[3] = uint8(units >> 8)
		setU32LE(c.flushBuf0[4:8], uint32(c.window))
		setU32LE(c.flushBuf0[8:12], uint32(c.gc))
		setU32LE(c.flushBuf0[12:16], 1<<16|uint32(w))
		c.flushBuf0[21] = 0x18 // depth = 24 bits.

		// Pix holds the image's pixels. The pixel at (x, y) is Pix[y*Stride+x].
		stride := c.bufimg.Stride
		for y := dirty.Min.Y; y < dirty.Max.Y; y++ {
			setU32LE(c.flushBuf0[16:20], uint32(y<<16|dirty.Min.X))
			_, err := c.w.Write(c.flushBuf0[0:24])
			if err != nil {
				c.flushLock.Unlock()
				return
			}
			row := c.bufimg.Pix[y*stride:]
			for x := dirty.Min.X; x < dirty.Max.X; {
				nx := dirty.Max.X - x
				if nx > len(c.flushBuf1)/4 {
					nx = len(c.flushBuf1) / 4
				}
				for i, rgba := range row[x : x+nx] {
					c.flushBuf1[4*i+0] = rgba.B
					c.flushBuf1[4*i+1] = rgba.G
					c.flushBuf1[4*i+2] = rgba.R
				}
				x += nx
				_, err := c.w.Write(c.flushBuf1[0 : 4*nx])
				if err != nil {
					c.flushLock.Unlock()
					return
				}
			}
		}
		if c.w.Flush() != nil {
			c.flushLock.Unlock()
			return
		}
		c.flushLock.Unlock()
	}
}

func (c *conn) Close() error {
	// TODO
	return nil
}

func (c *conn) Screen() draw.Image { return c.img }

func (c *conn) FlushImageRect(r image.Rectangle) {
	c.flushLock.Lock()
	draw.DrawMask(c.bufimg, r, c.img, r.Min, nil, image.ZP, draw.Src)
	c.dirty = c.dirty.Union(r)
	// We do the send (the <- operator) in an expression context, rather than in
	// a statement context, so that it does not block, and fails if the buffered
	// channel is full (in which case there already is a flush request pending).
	// We send with the lock held to avoid the flusher picking up our flush
	// notification after it has actually dealt with it.
	select {
	case c.flush <- false:
	default:
	}
	c.flushLock.Unlock()
}

func (c *conn) FlushImage() {
	c.FlushImageRect(c.img.Bounds())
}

func (c *conn) EventChan() <-chan interface{} {
	return c.event
}

// pumper runs in its own goroutine, reading X events and demuxing them over the kbd / mouse / resize / quit chans.
func (c *conn) pumper(mouse chan<- draw.MouseEvent) {
	var timestamp timeTranslate
	for {
		// X events are always 32 bytes long.
		_, err := io.ReadFull(c.r, c.buf[0:32])
		if err != nil {
			// TODO(nigeltao): should draw.Window expose err?
			// TODO(nigeltao): should we do c.quit<-true? Should c.quit be a buffered channel?
			// Or is c.quit only for non-exceptional closing (e.g. when the window manager destroys
			// our window), and not for e.g. an I/O error?
			break
		}
		switch c.buf[0] {
		case 0x02, 0x03: // Key press, key release.
			// BUG(nigeltao): Keycode to keysym mapping is not implemented.

			// The keycode is in c.buf[1], but as keymaps aren't implemented yet, we'll use the
			// space character as a placeholder.
			keysym := int(' ')
			// TODO(nigeltao): Should we send KeyboardChan ints for Shift/Ctrl/Alt? Should Shift-A send
			// the same int down the channel as the sent on just the A key?
			// TODO(nigeltao): How should IME events (e.g. key presses that should generate CJK text) work? Or
			// is that outside the scope of the draw.Window interface?
			if c.buf[0] == 0x03 {
				keysym = -keysym
			}
			c.event <- draw.KeyEvent{keysym}
		case 0x04, 0x05: // Button press, button release.
			c.mouseState.Nsec = timestamp.Nanoseconds(getU32LE(c.buf[4:8]))
			mask := 1 << (c.buf[1] - 1)
			if c.buf[0] == 0x04 {
				c.mouseState.Buttons |= mask
			} else {
				c.mouseState.Buttons &^= mask
			}
			mouse <- c.mouseState
		case 0x06: // Motion notify.
			c.mouseState.Nsec = timestamp.Nanoseconds(getU32LE(c.buf[4:8]))
			c.mouseState.Loc.X = int(int16(c.buf[25])<<8 | int16(c.buf[24]))
			c.mouseState.Loc.Y = int(int16(c.buf[27])<<8 | int16(c.buf[26]))
			// TODO(nigeltao): update mouseState's timestamp.
			mouse <- c.mouseState
		case 0x0c: // Expose.
			// TODO(nigeltao): Should we ignore the very first expose event? A freshly mapped window
			// will trigger expose, but until the first c.FlushImage call, there's probably nothing to
			// paint but black. For an 800x600 window, at 4 bytes per pixel, each repaint writes about
			// 2MB over the socket.
			x := int(c.buf[9])<<8 | int(c.buf[8])
			y := int(c.buf[11])<<8 | int(c.buf[10])
			w := int(c.buf[13])<<8 | int(c.buf[12])
			h := int(c.buf[15])<<8 | int(c.buf[14])
			c.flushLock.Lock()
			c.dirty = c.dirty.Union(image.Rect(x, y, x+w, x+h))
			select {
			case c.flush <- false:
			default:
			}
			c.flushLock.Unlock()
			// TODO(nigeltao): Should we listen to DestroyNotify (0x11) and ResizeRequest (0x19) events?
			// What about EnterNotify (0x07) and LeaveNotify (0x08)?
		}
	}
	close(c.event)
}

// connect connects to the X server given by the full X11 display name (e.g.
// ":12.0") and returns the connection as well as the portion of the full name
// that is the display number (e.g. "12").
// Examples:
//	connect(":1")                 // calls net.Dial("unix", "", "/tmp/.X11-unix/X1"), displayStr="1"
//	connect("/tmp/launch-123/:0") // calls net.Dial("unix", "", "/tmp/launch-123/:0"), displayStr="0"
//	connect("hostname:2.1")       // calls net.Dial("tcp", "", "hostname:6002"), displayStr="2"
//	connect("tcp/hostname:1.0")   // calls net.Dial("tcp", "", "hostname:6001"), displayStr="1"
func connect(display string) (conn net.Conn, displayStr string, err error) {
	colonIdx := strings.LastIndex(display, ":")
	if colonIdx < 0 {
		return nil, "", errors.New("bad display: " + display)
	}
	// Parse the section before the colon.
	var protocol, host, socket string
	if display[0] == '/' {
		socket = display[0:colonIdx]
	} else {
		if i := strings.LastIndex(display, "/"); i < 0 {
			// The default protocol is TCP.
			protocol = "tcp"
			host = display[0:colonIdx]
		} else {
			protocol = display[0:i]
			host = display[i+1 : colonIdx]
		}
	}
	// Parse the section after the colon.
	after := display[colonIdx+1:]
	if after == "" {
		return nil, "", errors.New("bad display: " + display)
	}
	if i := strings.LastIndex(after, "."); i < 0 {
		displayStr = after
	} else {
		displayStr = after[0:i]
	}
	displayInt, err := strconv.Atoi(displayStr)
	if err != nil || displayInt < 0 {
		return nil, "", errors.New("bad display: " + display)
	}
	// Make the connection.
	if socket != "" {
		conn, err = net.Dial("unix", socket+":"+displayStr)
	} else if host != "" {
		conn, err = net.Dial(protocol, host+":"+strconv.Itoa(6000+displayInt))
	} else {
		conn, err = net.Dial("unix", "/tmp/.X11-unix/X"+displayStr)
	}
	if err != nil {
		return nil, "", errors.New("cannot connect to " + display + ": " + err.Error())
	}
	return
}

// authenticate authenticates ourselves with the X server.
// displayStr is the "12" out of ":12.0".
func authenticate(w *bufio.Writer, displayStr string) error {
	key, value, err := readAuth(displayStr)
	if err != nil {
		return err
	}
	// Assume that the authentication protocol is "MIT-MAGIC-COOKIE-1".
	if len(key) != 18 || len(value) != 16 {
		return errors.New("unsupported Xauth")
	}
	// 0x006c means little-endian. 0x000b, 0x0000 means X major version 11, minor version 0.
	// 0x0012 and 0x0010 means the auth key and value have lenths 18 and 16.
	// The final 0x0000 is padding, so that the string length is a multiple of 4.
	_, err = io.WriteString(w, "\x6c\x00\x0b\x00\x00\x00\x12\x00\x10\x00\x00\x00")
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, key)
	if err != nil {
		return err
	}
	// Again, the 0x0000 is padding.
	_, err = io.WriteString(w, "\x00\x00")
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, value)
	if err != nil {
		return err
	}
	err = w.Flush()
	if err != nil {
		return err
	}
	return nil
}

// readU8 reads a uint8 from r, using b as a scratch buffer.
func readU8(r io.Reader, b []byte) (uint8, error) {
	_, err := io.ReadFull(r, b[0:1])
	if err != nil {
		return 0, err
	}
	return uint8(b[0]), nil
}

// readU16LE reads a little-endian uint16 from r, using b as a scratch buffer.
func readU16LE(r io.Reader, b []byte) (uint16, error) {
	_, err := io.ReadFull(r, b[0:2])
	if err != nil {
		return 0, err
	}
	return uint16(b[0]) | uint16(b[1])<<8, nil
}

// readU32LE reads a little-endian uint32 from r, using b as a scratch buffer.
func readU32LE(r io.Reader, b []byte) (uint32, error) {
	_, err := io.ReadFull(r, b[0:4])
	if err != nil {
		return 0, err
	}
	return getU32LE(b), nil
}

// Gets the little-endian representation of u from b[0:4]
func getU32LE(b []byte) (u uint32) {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

// setU32LE sets b[0:4] to be the little-endian representation of u.
func setU32LE(b []byte, u uint32) {
	b[0] = byte((u >> 0) & 0xff)
	b[1] = byte((u >> 8) & 0xff)
	b[2] = byte((u >> 16) & 0xff)
	b[3] = byte((u >> 24) & 0xff)
}

// checkPixmapFormats checks that we have an agreeable X pixmap Format.
func checkPixmapFormats(r io.Reader, b []byte, n int) (agree bool, err error) {
	for i := 0; i < n; i++ {
		_, err = io.ReadFull(r, b[0:8])
		if err != nil {
			return
		}
		// Byte 0 is depth, byte 1 is bits-per-pixel, byte 2 is scanline-pad, the rest (5) is padding.
		if b[0] == 24 && b[1] == 32 {
			agree = true
		}
	}
	return
}

// checkDepths checks that we have an agreeable X Depth (i.e. one that has an agreeable X VisualType).
func checkDepths(r io.Reader, b []byte, n int, visual uint32) (agree bool, err error) {
	for i := 0; i < n; i++ {
		depth, err := readU16LE(r, b)
		if err != nil {
			return
		}
		depth &= 0xff
		visualsLen, err := readU16LE(r, b)
		if err != nil {
			return
		}
		// Ignore 4 bytes of padding.
		_, err = io.ReadFull(r, b[0:4])
		if err != nil {
			return
		}
		for j := 0; j < int(visualsLen); j++ {
			// Read 24 bytes: visual(4), class(1), bits per rgb value(1), colormap entries(2),
			// red mask(4), green mask(4), blue mask(4), padding(4).
			v, err := readU32LE(r, b)
			_, err = readU32LE(r, b)
			rm, err := readU32LE(r, b)
			gm, err := readU32LE(r, b)
			bm, err := readU32LE(r, b)
			_, err = readU32LE(r, b)
			if err != nil {
				return
			}
			if v == visual && rm == 0xff0000 && gm == 0xff00 && bm == 0xff && depth == 24 {
				agree = true
			}
		}
	}
	return
}

// checkScreens checks that we have an agreeable X Screen.
func checkScreens(r io.Reader, b []byte, n int) (root, visual uint32, err error) {
	for i := 0; i < n; i++ {
		root0, err := readU32LE(r, b)
		if err != nil {
			return
		}
		// Ignore the next 7x4 bytes, which is: colormap, whitepixel, blackpixel, current input masks,
		// width and height (pixels), width and height (mm), min and max installed maps.
		_, err = io.ReadFull(r, b[0:28])
		if err != nil {
			return
		}
		visual0, err := readU32LE(r, b)
		if err != nil {
			return
		}
		// Next 4 bytes: backing stores, save unders, root depth, allowed depths length.
		x, err := readU32LE(r, b)
		if err != nil {
			return
		}
		nDepths := int(x >> 24)
		agree, err := checkDepths(r, b, nDepths, visual0)
		if err != nil {
			return
		}
		if agree && root == 0 {
			root = root0
			visual = visual0
		}
	}
	return
}

// handshake performs the protocol handshake with the X server, and ensures
// that the server provides a compatible Screen, Depth, etc.
func (c *conn) handshake() error {
	_, err := io.ReadFull(c.r, c.buf[0:8])
	if err != nil {
		return err
	}
	// Byte 0:1 should be 1 (success), bytes 2:6 should be 0xb0000000 (major/minor version 11.0).
	if c.buf[0] != 1 || c.buf[2] != 11 || c.buf[3] != 0 || c.buf[4] != 0 || c.buf[5] != 0 {
		return errors.New("unsupported X version")
	}
	// Ignore the release number.
	_, err = io.ReadFull(c.r, c.buf[0:4])
	if err != nil {
		return err
	}
	// Read the resource ID base.
	resourceIdBase, err := readU32LE(c.r, c.buf[0:4])
	if err != nil {
		return err
	}
	// Read the resource ID mask.
	resourceIdMask, err := readU32LE(c.r, c.buf[0:4])
	if err != nil {
		return err
	}
	if resourceIdMask < 256 {
		return errors.New("X resource ID mask is too small")
	}
	// Ignore the motion buffer size.
	_, err = io.ReadFull(c.r, c.buf[0:4])
	if err != nil {
		return err
	}
	// Read the vendor length.
	vendorLen, err := readU16LE(c.r, c.buf[0:2])
	if err != nil {
		return err
	}
	if vendorLen != 20 {
		// For now, assume the vendor is "The X.Org Foundation". Supporting different
		// vendors would require figuring out how much padding we need to read.
		return errors.New("unsupported X vendor")
	}
	// Read the maximum request length.
	maxReqLen, err := readU16LE(c.r, c.buf[0:2])
	if err != nil {
		return err
	}
	if maxReqLen != 0xffff {
		return errors.New("unsupported X maximum request length")
	}
	// Read the roots length.
	rootsLen, err := readU8(c.r, c.buf[0:1])
	if err != nil {
		return err
	}
	// Read the pixmap formats length.
	pixmapFormatsLen, err := readU8(c.r, c.buf[0:1])
	if err != nil {
		return err
	}
	// Ignore some things that we don't care about (totalling 30 bytes):
	// imageByteOrder(1), bitmapFormatBitOrder(1), bitmapFormatScanlineUnit(1) bitmapFormatScanlinePad(1),
	// minKeycode(1), maxKeycode(1), padding(4), vendor(20, hard-coded above).
	_, err = io.ReadFull(c.r, c.buf[0:30])
	if err != nil {
		return err
	}
	// Check that we have an agreeable pixmap format.
	agree, err := checkPixmapFormats(c.r, c.buf[0:8], int(pixmapFormatsLen))
	if err != nil {
		return err
	}
	if !agree {
		return errors.New("unsupported X pixmap formats")
	}
	// Check that we have an agreeable screen.
	root, visual, err := checkScreens(c.r, c.buf[0:24], int(rootsLen))
	if err != nil {
		return err
	}
	if root == 0 || visual == 0 {
		return errors.New("unsupported X screen")
	}
	c.gc = resID(resourceIdBase)
	c.window = resID(resourceIdBase + 1)
	c.root = resID(root)
	c.visual = resID(visual)
	return nil
}

// NewWindow calls NewWindowDisplay with $DISPLAY.
func NewWindow() (draw.Window, error) {
	display := os.Getenv("DISPLAY")
	if len(display) == 0 {
		return nil, errors.New("$DISPLAY not set")
	}
	return NewWindowDisplay(display)
}

// NewWindowDisplay returns a new draw.Context, backed by a newly created and
// mapped X11 window. The X server to connect to is specified by the display
// string, such as ":1".
func NewWindowDisplay(display string) (draw.Window, error) {
	socket, displayStr, err := connect(display)
	if err != nil {
		return nil, err
	}
	c := new(conn)
	c.c = socket
	c.r = bufio.NewReader(socket)
	c.w = bufio.NewWriter(socket)
	err = authenticate(c.w, displayStr)
	if err != nil {
		return nil, err
	}
	err = c.handshake()
	if err != nil {
		return nil, err
	}

	// Now that we're connected, show a window, via three X protocol messages.
	// First, create a graphics context (GC).
	setU32LE(c.buf[0:4], 0x00060037) // 0x37 is the CreateGC opcode, and the message is 6 x 4 bytes long.
	setU32LE(c.buf[4:8], uint32(c.gc))
	setU32LE(c.buf[8:12], uint32(c.root))
	setU32LE(c.buf[12:16], 0x00010004) // Bit 2 is XCB_GC_FOREGROUND, bit 16 is XCB_GC_GRAPHICS_EXPOSURES.
	setU32LE(c.buf[16:20], 0x00000000) // The Foreground is black.
	setU32LE(c.buf[20:24], 0x00000000) // GraphicsExposures' value is unused.
	// Second, create the window.
	setU32LE(c.buf[24:28], 0x000a0001) // 0x01 is the CreateWindow opcode, and the message is 10 x 4 bytes long.
	setU32LE(c.buf[28:32], uint32(c.window))
	setU32LE(c.buf[32:36], uint32(c.root))
	setU32LE(c.buf[36:40], 0x00000000) // Initial (x, y) is (0, 0).
	setU32LE(c.buf[40:44], windowHeight<<16|windowWidth)
	setU32LE(c.buf[44:48], 0x00010000) // Border width is 0, XCB_WINDOW_CLASS_INPUT_OUTPUT is 1.
	setU32LE(c.buf[48:52], uint32(c.visual))
	setU32LE(c.buf[52:56], 0x00000802) // Bit 1 is XCB_CW_BACK_PIXEL, bit 11 is XCB_CW_EVENT_MASK.
	setU32LE(c.buf[56:60], 0x00000000) // The Back-Pixel is black.
	setU32LE(c.buf[60:64], 0x0000804f) // Key/button press and release, pointer motion, and expose event masks.
	// Third, map the window.
	setU32LE(c.buf[64:68], 0x00020008) // 0x08 is the MapWindow opcode, and the message is 2 x 4 bytes long.
	setU32LE(c.buf[68:72], uint32(c.window))
	// Write the bytes.
	_, err = c.w.Write(c.buf[0:72])
	if err != nil {
		return nil, err
	}
	err = c.w.Flush()
	if err != nil {
		return nil, err
	}

	c.img = image.NewRGBA(image.Rect(0, 0, windowWidth, windowHeight))
	c.bufimg = image.NewRGBA(image.Rect(0, 0, windowWidth, windowHeight))
	// TODO(nigeltao): Should these channels be buffered?
	c.event = make(chan interface{})
	mouse := make(chan draw.MouseEvent)
	go bufferMouse(mouse, c.event)
	c.flush = make(chan bool, 1)
	go c.flusher()
	go c.pumper(mouse)
	return c, nil
}

// timeTranslate translates from milliseconds to nanoseconds
// since the epoch.
//
// The first time a translation is done, it uses the current time
// to find an absolute time. This means that if delivery of the
// first event is delayed, then all timestamps will be off,
// but generally it's only the deltas that are important for
// mouse events, and there's not much else we can do.
//
type timeTranslate struct {
	t0  int64  // absolute time of first event.
	ms0 uint32 // millisecond time of first event.
}

func (t *timeTranslate) Nanoseconds(ms uint32) int64 {
	if t.t0 == 0 {
		t.ms0 = ms
		t.t0 = time.Now()
	}
	return int64(ms-t.ms0)*1e6 + t.t0
}

func bufferMouse(mc <-chan draw.MouseEvent, out chan<- interface{}) {
	type mouseQueue struct {
		m    draw.MouseEvent
		next *mouseQueue
	}
	actualOut := out
	q := (*mouseQueue)(nil)
	eq := &q
	eof := false
	var state draw.MouseEvent
	for {
		// Try to send an event if there are any events in the queue
		if q != nil {
			state = q.m
			out = actualOut
		} else {
			out = nil
			if eof {
				close(out)
				return
			}
		}
		select {
		case m := <-mc:
			if closed(mc) {
				// When the in channel is closed, make
				// sure that all events drain from the
				// queue before closing it.
				// This may be considered unnecessary.
				eof = true
				mc = nil
				break
			}

			// Only if the queue is empty or the buttons state
			// has changed do we add a new event to the queue;
			// otherwise we just update the event at its head.
			if q == nil || m.Buttons != state.Buttons {
				*eq = &mouseQueue{m, nil}
				eq = &(*eq).next
			} else {
				q.m = m
			}
		case out <- state:
			if q = q.next; q == nil {
				eq = &q
			}
		}
	}
}
