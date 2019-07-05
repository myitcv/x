package gogenerate

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
)

const (
	archiveDelim   = byte('|')
	archiveVersion = byte('1')
)

type archiveReader struct {
	file        *os.File
	und         *bufio.Reader
	readVersion bool
}

func newArchiveReader(path string) (*archiveReader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %v: %v", path, err)
	}
	return &archiveReader{
		file: f,
		und:  bufio.NewReader(f),
	}, nil
}

func (r *archiveReader) ExtractFile() (string, error) {
	if !r.readVersion {
		b, err := r.und.ReadByte()
		if err == nil {
			if b != archiveVersion {
				err = fmt.Errorf("read version %v; expected %v", string(b), string(archiveVersion))
			}
			r.readVersion = true
		}
		if err != nil {
			return "", err
		}
	}
	fn, err := r.und.ReadString(archiveDelim)
	if err != nil {
		if err == io.EOF {
			return "", err
		}
		return "", fmt.Errorf("could not read filename: %v", err)
	}
	fn = fn[:len(fn)-1]
	if len(fn) == 0 {
		return "", fmt.Errorf("invalid zero-length filename")
	}
	ls, err := r.und.ReadString(archiveDelim)
	if err != nil {
		return "", fmt.Errorf("failed to read length string: %v", err)
	}
	ls = ls[:len(ls)-1]
	l, err := strconv.ParseInt(ls, 10, 64)
	if err != nil {
		return "", fmt.Errorf("failed to read length of file: %v", err)
	}
	f, err := os.Create(fn)
	if err != nil {
		return "", fmt.Errorf("failed to create %v: %v", fn, err)
	}
	lr := io.LimitReader(r.und, l)
	if n, err := io.Copy(f, lr); err != nil || n != l {
		return "", fmt.Errorf("failed to extract %v bytes (read %v) to %v: %v", l, n, fn, err)
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("failed to close %v: %v", fn, err)
	}
	return fn, nil
}

func (r *archiveReader) Close() error {
	return r.file.Close()
}

type archiveWriter struct {
	file           *os.File
	und            *bufio.Writer
	writtenVersion bool
}

func newArchiveWriter(dir string, pattern string) (*archiveWriter, error) {
	tf, err := ioutil.TempFile(dir, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %v", err)
	}
	return &archiveWriter{
		file: tf,
		und:  bufio.NewWriter(tf),
	}, nil
}

func (w *archiveWriter) PutFile(path string) error {
	if !w.writtenVersion {
		if err := w.und.WriteByte(archiveVersion); err != nil {
			return fmt.Errorf("failed to write archive version: %v", err)
		}
		w.writtenVersion = true
	}
	if _, err := w.und.WriteString(path + string(archiveDelim)); err != nil {
		return fmt.Errorf("failed to write file path %v: %v", path, err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat %v: %v", path, err)
	}
	if _, err := w.und.WriteString(fmt.Sprintf("%v%v", fi.Size(), string(archiveDelim))); err != nil {
		return fmt.Errorf("failed to write file length")
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open %v for reading: %v", path, err)
	}
	n, err := io.Copy(w.und, f)
	f.Close()
	if err != nil || n != fi.Size() {
		return fmt.Errorf("failed to write %v to archive: wrote %v (expected %v): %v", path, n, fi.Size(), err)
	}
	return nil
}

func (w *archiveWriter) Close() error {
	if err := w.und.Flush(); err != nil {
		return fmt.Errorf("failed to flush buffer: %v", err)
	}
	return w.file.Close()
}
