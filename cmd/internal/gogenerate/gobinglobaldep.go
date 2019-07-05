package gogenerate

import "fmt"

type gobinGlobalDep struct {
	*commandDep

	targetPath string
}

func (g *gobinGlobalDep) String() string {
	return fmt.Sprintf("gobinGlobalDep: %v", g.targetPath)
}

func (g *gobinGlobalDep) HashString() string {
	return fmt.Sprintf("gobinGlobalDep: %v [%x]", g.targetPath, g.hash)
}

var _ generator = (*gobinGlobalDep)(nil)
