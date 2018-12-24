package main

import "fmt"

type commandDep struct {
	name string
	hash [hashSize]byte

	depsMap
}

func (c *commandDep) Deps() depsMap {
	return c.depsMap
}

func (c *commandDep) Ready() bool {
	return true
}

func (c *commandDep) Done() bool {
	return c.Ready() && c.hash != nilHash
}

func (c *commandDep) Undo() {
	c.hash = nilHash
}

func (c *commandDep) String() string {
	return fmt.Sprintf("commandDep: %v", c.name)
}

func (c *commandDep) HashString() string {
	return fmt.Sprintf("commandDep: %v [%x]", c.name, c.hash)
}

func (c *commandDep) DirectiveName() string {
	return c.name
}

var _ generator = (*commandDep)(nil)
