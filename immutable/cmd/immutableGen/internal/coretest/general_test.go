package coretest

import (
	"testing"
	"time"

	"myitcv.io/immutable/cmd/immutableGen/internal/coretest/pkga"
	"myitcv.io/immutable/cmd/immutableGen/internal/coretest/pkgb"
)

func TestAnonFields(t *testing.T) {
	m := new(MyStruct)

	if v := m.string(); v != "" {
		t.Fatalf("expected zero value to be %v", "")
	}

	val := "test"
	m = m.setString(val)

	if v := m.string(); v == "" || v != val {
		t.Fatalf("expected set value to be %q", val)
	}
}

func TestEmbedAccess(t *testing.T) {
	now := time.Now()
	ns := NonImmStruct{
		Now: now,
	}
	b := new(pkgb.PkgB).SetPostcode("London")
	c1 := new(Clash1).SetNoClash1("NoClash1")
	c2 := new(pkga.Clash2).SetNoClash2("NoClash2")
	a := new(pkga.PkgA).SetAddress("home").SetPkgB(b)
	e2 := new(Embed2).SetAge(42)
	e1 := new(Embed1).WithMutable(func(e1 *Embed1) {
		e1.SetName("Paul")
		e1.SetEmbed2(e2)
		e1.SetPkgA(a)
		e1.SetClash1(c1)
		e1.SetClash2(c2)
		e1.SetNonImmStruct(ns)
	})

	{
		want := 42
		if got := e2.Age(); want != got {
			t.Fatalf("e2.Age(): want %v, got %v", want, got)
		}
	}
	{
		want := 42
		if got := e1.Age(); want != got {
			t.Fatalf("e1.Age(): want %v, got %v", want, got)
		}
	}
	{
		want := "home"
		if got := e1.Address(); want != got {
			t.Fatalf("e1.Address(): want %v, got %v", want, got)
		}
	}
	{
		want := "NoClash1"
		if got := e1.NoClash1(); want != got {
			t.Fatalf("e1.NoClash1(): want %v, got %v", want, got)
		}
	}
	{
		want := "NoClash2"
		if got := e1.NoClash2(); want != got {
			t.Fatalf("e1.NoClash2(): want %v, got %v", want, got)
		}
	}
	{
		want := "London"
		if got := e1.Postcode(); want != got {
			t.Fatalf("e1.Postcode(): want %v, got %v", want, got)
		}
	}
	{
		want := now
		if got := e1.Now(); want != got {
			t.Fatalf("e1.Now(): want %v, got %v", want, got)
		}
	}

	newNow := time.Now()
	e1p := e1.SetNow(newNow)

	{
		want := now
		if got := e1.Now(); want != got {
			t.Fatalf("e1.Now(): want %v, got %v", want, got)
		}
	}
	{
		want := newNow
		if got := e1p.Now(); want != got {
			t.Fatalf("e2.Now(): want %v, got %v", want, got)
		}
	}
}
