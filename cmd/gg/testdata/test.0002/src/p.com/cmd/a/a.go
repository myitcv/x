package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

func main() {
	p := os.Getenv("GOPACKAGE")
	fn := fmt.Sprintf("gen_%v_a.go", p)
	fc := fmt.Sprintf("package %v\n", p)
	ioutil.WriteFile(fn, []byte(fc), 0644)
}
