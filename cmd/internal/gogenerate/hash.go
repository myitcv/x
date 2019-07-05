package gogenerate

import (
	"bytes"
	"io"

	"github.com/rogpeppe/go-internal/cache"
)

type hash struct {
	h        *cache.Hash
	hw       io.Writer
	debugOut *bytes.Buffer
}

func (h *hash) Write(p []byte) (n int, err error) {
	return h.hw.Write(p)
}

func (h *hash) String() string {
	return h.debugOut.String()
}

func (h *hash) Sum() [hashSize]byte {
	return h.h.Sum()
}

func newHash(name string) *hash {
	res := &hash{
		h: cache.NewHash(name),
	}
	if hashDebug {
		res.debugOut = new(bytes.Buffer)
		res.hw = io.MultiWriter(res.debugOut, res.h)
	} else {
		res.hw = res.h
	}
	return res
}
