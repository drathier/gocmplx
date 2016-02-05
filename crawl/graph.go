package crawl

import (
	"sync"
	"fmt"
)

// struct used to store dependency graph, which is required to be a directed acyclic graph

type Package struct {
	m sync.RWMutex
	Deps []*Package // make a method instead?
	IsStdlib bool
	files []*File
	path string
}

type File struct {
	Usages map[string]*Package // ast representation of type -> package

	// where
	src string
	row int
	col int
}

// Dependencies calculates what things in this package that references what things in other packages
func (g *Package) Dependencies() {

}

func (g *Package) Search(path string) (*Package, error) {
	if g.path == path {
		return g, nil
	}
	for _, pkg := range g.Deps {
		if p, err := pkg.Search(path); err == nil {
			return p
		}
	}
	return nil, fmt.Errorf("package not found")
}

// ParsePackage returns a pointer to the topmost package in the newly created package graph
func ParsePackage(path string) *Package {

}