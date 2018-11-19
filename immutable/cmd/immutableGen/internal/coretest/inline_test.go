package coretest

import (
	"testing"

	"github.com/Quasilyte/inltest"
)

func TestWrapperInlines(t *testing.T) {
	issues, err := inltest.CheckInlineable(map[string][]string{
		"myitcv.io/immutable/cmd/immutableGen/internal/coretest": {
			// TODO complete this list based on current expectations
			"(*MyMap).Get",
			"(*MyStruct).Key",
			"(*MyStruct).SetKey",
		},
	})

	if err != nil {
		t.Fatalf("failed to check inlineability: %v", err)
	}

	if len(issues) != 0 {
		t.Fatalf("unexpected ")
	}
}
