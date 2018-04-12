package vgo_test

import (
	"os"
	"testing"

	// import a non-standard library package for its side effects.
	// vgo will then detect this
	_ "golang.org/x/net/html"
	"myitcv.io/vgo"
)

func TestLoader(t *testing.T) {
	// given the side-effect import above, we can now create a Loader
	// to load "golang.org/x/net/html" in the context of the current
	// directory

	l := vgo.NewTestLoader(".")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", cwd)
	}

	cases := []string{
		"golang.org/x/net/html",

		// this is a dependency of x/net/html; hence an indirect
		// test dependency of vgo_test
		"golang.org/x/net/html/atom",
	}

	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			p, err := l.ImportFrom(c, cwd, 0)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if p == nil {
				t.Fatal("expected response; got nil")
			}

			if v := p.Path(); v != c {
				t.Fatalf("expected ImportPath %q; got %q", c, v)
			}
		})
	}
}
