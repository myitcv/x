package main

import (
	"testing"
)

func TestFoldOnSpaces(t *testing.T) {
	w := 20

	checks := []struct {
		i string
		o string
	}{
		{
			i: "\t\ttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttt",
			o: "\t\ttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttt",
		},
		{
			i: "this is a test",
			o: "this is a test",
		},
		{
			i: "the quick brown fox jumped over the lazy sleeping dog",
			o: "the quick brown fox\n" +
				"jumped over the\n" +
				"lazy sleeping dog",
		},
		{
			i: "\t\tthe quick brown fox jumped over the lazy sleeping dog",
			o: "\t\tthe quick brown\n" +
				"\t\tfox jumped over\n" +
				"\t\tthe lazy sleeping\n" +
				"\t\tdog",
		},
		{
			i: `Flags:
		-outpkgname string
		name of package to output; if not specified take the package name of the input directory/import path`,
			o: "Flags:\n" +
				"\t\t-outpkgname\n" +
				"\t\tstring\n" +
				"\t\tname of package\n" +
				"\t\tto output; if not\n" +
				"\t\tspecified take\n" +
				"\t\tthe package name\n" +
				"\t\tof the input\n" +
				"\t\tdirectory/import\n" +
				"\t\tpath",
		},
	}

	for _, c := range checks {
		out := c.o

		res := foldOnSpaces(c.i, w)
		if res != out {
			t.Errorf("%q failed to fold to\n%q; got\n%q", c.i, out, res)
		}
	}
}
