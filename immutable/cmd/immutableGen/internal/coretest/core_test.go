package coretest_test

import (
	_ "myitcv.io/immutable/cmd/immutableGen/internal/coretest"
)

//go:generate gobin -m -run myitcv.io/immutable/cmd/immutableGen -licenseFile license.txt -G "echo \"hello world\""

type _Imm_xtestA struct {
	*XTestB

	age int
}

type _Imm_XTestB struct {
	Name string
}
