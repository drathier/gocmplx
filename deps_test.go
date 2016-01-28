package main

import (
	"testing"
)

func TestDeps(t *testing.T) {
	findDeps("github.com/drathier/saiph/odb", "github.com/drathier/saiph/odb/oauth")
}
