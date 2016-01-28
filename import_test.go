package main

import (
	"fmt"
	"testing"
)

func TestImport(t *testing.T) {
	importPkg("go/build")
	fmt.Println(depmap)
}
