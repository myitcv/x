// +build js

package main_test

import (
	"os"
	"testing"
)

const (
	envVar = "BANANA"
)

func Test(t *testing.T) {
	want := "banana"
	if got := os.Getenv(envVar); got != want {
		t.Fatalf("expected env %v to be %v; got %v", envVar, want, got)
	}
}
