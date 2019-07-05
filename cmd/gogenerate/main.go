package main

import "myitcv.io/cmd/internal/gogenerate"

//go:generate gobin -m -run myitcv.io/cmd/helpflagtopkgdoc

func main() {
	gogenerate.Main()
}
