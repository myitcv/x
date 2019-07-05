package main

import (
	"fmt"
	"os"

	"myitcv.io/cmd/internal/gogenerate"
)

func main() {
	fmt.Fprintf(os.Stderr, "gg is now gogenerate\n")
	gogenerate.Main()
}
