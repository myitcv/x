// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package x11

import (
	"bufio"
	"errors"
	"io"
	"os"
)

// readU16BE reads a big-endian uint16 from r, using b as a scratch buffer.
func readU16BE(r io.Reader, b []byte) (uint16, error) {
	_, err := io.ReadFull(r, b[0:2])
	if err != nil {
		return 0, err
	}
	return uint16(b[0])<<8 + uint16(b[1]), nil
}

// readStr reads a length-prefixed string from r, using b as a scratch buffer.
func readStr(r io.Reader, b []byte) (string, error) {
	n, err := readU16BE(r, b)
	if err != nil {
		return "", err
	}
	if int(n) > len(b) {
		return "", errors.New("Xauthority entry too long for buffer")
	}
	_, err = io.ReadFull(r, b[0:n])
	if err != nil {
		return "", err
	}
	return string(b[0:n]), nil
}

// readAuth reads the X authority file and returns the name/data pair for the display.
// displayStr is the "12" out of a $DISPLAY like ":12.0".
func readAuth(displayStr string) (name, data string, err error) {
	// b is a scratch buffer to use and should be at least 256 bytes long
	// (i.e. it should be able to hold a hostname).
	var b [256]byte
	// As per /usr/include/X11/Xauth.h.
	const familyLocal = 256

	fn := os.Getenv("XAUTHORITY")
	if fn == "" {
		home := os.Getenv("HOME")
		if home == "" {
			return "", "", errors.New("Xauthority not found: $XAUTHORITY, $HOME not set")

		}
		fn = home + "/.Xauthority"
	}
	r, err := os.Open(fn)
	if err != nil {
		return "", "", err
	}
	defer r.Close()
	br := bufio.NewReader(r)

	hostname, err := os.Hostname()
	if err != nil {
		return "", "", err
	}
	for {
		family, err := readU16BE(br, b[0:2])
		if err != nil {
			return "", "", err
		}
		addr, err := readStr(br, b[0:])
		if err != nil {
			return "", "", err
		}
		disp, err := readStr(br, b[0:])
		if err != nil {
			return "", "", err
		}
		name0, err := readStr(br, b[0:])
		if err != nil {
			return "", "", err
		}
		data0, err := readStr(br, b[0:])
		if err != nil {
			return "", "", err
		}
		if family == familyLocal && addr == hostname && disp == displayStr {
			return name0, data0, nil
		}
	}
	panic("unreachable")
}
