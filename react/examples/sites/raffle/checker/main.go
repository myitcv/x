// checker is a simpler command line tool for checking entries to the raffle
package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

var (
	fKey = flag.String("key", "", "the secret key")
)

// Takes either handle:entry pairs args command line args, or if there are none
// read lines from stdin

func main() {
	flag.Parse()

	if *fKey == "" {
		fmt.Fprintf(os.Stderr, "must provide -key\n")
		os.Exit(2)
	}

	var entries []string
	if len(flag.Args()) > 0 {
		entries = flag.Args()
	} else {
		in, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read stdin: %v\n", err)
			os.Exit(1)
		}
		entries = strings.Split(string(in), "\n")
	}

	for _, pair := range entries {
		pair := strings.TrimSpace(pair)
		if pair == "" {
			// blank line from stdin?
			continue
		}
		parts := strings.Split(pair, ":")
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "invalid entry %q\n", pair)
			os.Exit(1)
		}
		handle, entry := parts[0], parts[1]
		if check(handle, entry) {
			fmt.Printf("%v is a winner!\n", handle)
		}
	}
}

func check(handle, entry string) bool {
	hash := sha256.New()
	fmt.Fprintf(hash, "Handle: %v\n", handle)
	fmt.Fprintf(hash, "Key: %v\n", *fKey)

	valid := fmt.Sprintf("%x", hash.Sum(nil))
	return valid == entry
}
