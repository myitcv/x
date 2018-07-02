// Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
// Use of this document is governed by a license found in the LICENSE document.

package main // import "myitcv.io/cmd/compressHistory"

import (
	"fmt"
	"os"

	"github.com/rogpeppe/rog-go/reverse"
)

// assumes that files have been provided in the order oldest to newset
// read files, last first, end-to-beginning, taking unique history
// entries (i.e. most recent history entries considered first)
//
// output the reverse of that... i.e. oldest first, newest last

func main() {
	x := os.Args[1:]

	var out []string
	seen := make(map[string]struct{})

	for i := len(x) - 1; i >= 0; i-- {
		fn := x[i]
		fh, err := os.Open(fn)
		if err != nil {
			panic(fmt.Sprintf("Could not open %v", fn))
		}
		scanner := reverse.NewScanner(fh)
		for scanner.Scan() {
			if _, ok := seen[scanner.Text()]; !ok {
				seen[scanner.Text()] = struct{}{}
				out = append(out, scanner.Text())
			}
		}
		err = fh.Close()
		if err != nil {
			panic(fmt.Sprintf("Could not close %v", fn))
		}
	}

	for i := len(out) - 1; i >= 0; i-- {
		fmt.Println(out[i])
	}
}
