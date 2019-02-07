package main

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestArchive(t *testing.T) {
	td, err := ioutil.TempDir("", "gg-TestArchive")
	if err != nil {
		t.Fatalf("failed to create TempDir: %v", err)
	}
	defer os.RemoveAll(td)

	os.Mkdir(filepath.Join(td, "workings"), 0777)

	files := []struct {
		path     string
		contents string
	}{
		{filepath.Join(td, "a"), "this is a"},
		{filepath.Join(td, "b"), "this is b"},
	}
	for _, f := range files {
		if err := ioutil.WriteFile(f.path, []byte(f.contents), 0666); err != nil {
			t.Fatalf("failed to write file %v: %v", f.path, err)
		}
	}
	w, err := newArchiveWriter(filepath.Join(td, "workings"), "")
	if err != nil {
		t.Fatalf("failed to create writer: %v", err)
	}
	for _, f := range files {
		if err := w.PutFile(f.path); err != nil {
			t.Fatalf("failed to put file %v: %v", f.path, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}
	for _, f := range files {
		if err := os.Remove(f.path); err != nil {
			t.Fatalf("failed to remove %v: %v", f.path, err)
		}
	}
	r, err := newArchiveReader(w.file.Name())
	if err != nil {
		t.Fatalf("failed to create reader: %v", err)
	}
	for _, f := range files {
		fn, err := r.ExtractFile()
		if err != nil {
			t.Fatalf("failed to extract file: %v", err)
		}
		if fn != f.path {
			t.Fatalf("failed to extract correct file name; got %v, want %v", fn, f.path)
		}
		fc, err := ioutil.ReadFile(f.path)
		if err != nil {
			t.Fatalf("failed to read back file %v: %v", f.path, err)
		}
		if f.contents != string(fc) {
			t.Fatalf("mistmatch of contents in %v:\n%q\n%q\n", f.path, f.contents, string(fc))
		}
	}

	if _, err := r.ExtractFile(); err != io.EOF {
		t.Fatalf("expected io.EOF; got %v", err)
	}

	if err := r.Close(); err != nil {
		t.Fatalf("failed to close reader: %v", err)
	}
}
