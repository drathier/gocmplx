package main

import (
	"fmt"
	"testing"
)

func TestDeps(t *testing.T) {
	a := findDeps("github.com/drathier/saiph/odb", "github.com/drathier/saiph/odb/oauth")
	fmt.Printf("%#v\n", a)

	b := findDeps("github.com/drathier/saiph", "github.com/drathier/saiph/odb")
	fmt.Printf("%#v\n", b)
}
