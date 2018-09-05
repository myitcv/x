package pkga

import (
	"time"

	"myitcv.io/immutable/cmd/immutableGen/internal/coretest/pkgb"
)

//go:generate immutableGen

type _Imm_PkgA struct {
	*pkgb.PkgB
	Address string
}

type _Imm_Clash2 struct {
	Clash    string
	NoClash2 string
}

type _Imm_OtherA struct {
	OtherNameA string
}

type NonImmStructA struct {
	NowA time.Time
	*OtherA
}
