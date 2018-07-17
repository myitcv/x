// +build js

package main_test

import (
	"testing"

	"github.com/gopherjs/gopherjs/js"
)

func Test006(t *testing.T) {
	var x *js.Object
	x.Call("hello")
}
