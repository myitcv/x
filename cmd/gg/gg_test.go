package main_test

import (
	"testing"
)

func TestBasic(t *testing.T) {
	tg := testgg(t, "test.0001")
	defer tg.teardown()

	tg.clean()
	tg.setdir(tg.pd())
	tg.run("p.com")
	tg.ensure(tg.pd())
}

func TestNonCmd(t *testing.T) {
	tg := testgg(t, "test.0002")
	defer tg.teardown()

	tg.clean()
	tg.setdir(tg.pd())
	tg.run("p.com")
	tg.ensure(tg.pd())
}
