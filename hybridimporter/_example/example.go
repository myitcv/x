package example

import "fmt"

// this file does not type-check

type Blah = Test

const Name = "test"

func AFunc() string {
	return fmt.Sprintf("%v", asdf)
}
