package crawl

import (
	"testing"
	"fmt"
)

func TestImportPkg(t *testing.T) {
	depmap := make(map[string][]string)
	task := Task{}
	importPkg(depmap, "github.com/drathier/gocmplx", task, false)

	for k, vs := range depmap {
		fmt.Println(k, "->")
		for _, v := range vs {
			fmt.Printf("\t%#v\n", v)
		}
	}
}
