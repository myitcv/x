// +build ignore

package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"
)

const (
	iterations = 5
)

func main() {
	for i := 1; i <= iterations; i++ {
		time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
		fmt.Printf("Instance %v iteration loop %v\n", os.Args[1], i)
	}
}
