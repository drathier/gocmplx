package main

import (
	"bytes"
	"fmt"
	"go/build"
	"io/ioutil"
	"strings"
	"os"
)

var (
	deps = make(map[string][]string) // depends on
)

func importPkg(path string) {
	if _, found := deps[path]; found {
		return // already imported this
	}
	pkgs, err := build.Import(path, "", 0)
	if err != nil {
		panic(err)
	}
	for _, pkg := range pkgs.Imports {
		deps[path] = append(deps[path], pkg)
		importPkg(pkg)
	}
}

type dep struct {
	from string // current file path
	to   string // oracle describe/package/path
	typ  string // oracle describe/package/members[i]/type
}

// saiph.User depends on gocmplx.Deps; return filename.go:line:col imports filename2.go:line:col type struct{asd string; potato int}
func findDeps(path, pkgPath string) {
	// 1. get package name from pkgPath
	pkgName := pkgIdent(pkgPath)
	fmt.Println("pkgName", pkgName)

	// 2. find that package name in source files
	pkg, err := build.Import(path, "", 0)
	if err != nil {
		panic(err)
	}

	for _, filename := range pkg.GoFiles {
		fmt.Println("pkg.GoFiles", filename)
		file, err := ioutil.ReadFile(os.Getenv("GOPATH") + "/src/" + pkg.ImportPath + "/" + filename)
		if err != nil {
			panic(err)
		}
		for _, pos := range indexAll(file, []byte(pkgName)) {
			oracleLookup(pos, filename, path)
		}
	}
	// 3. check if those matches are in fact pointing to the package
	// 4. add things to result

}

func oracleLookup(pos int, file string, pkg string) {
	fmt.Printf("oracleLookup(pos %d, file %s, pkg %s)\n", pos, file, pkg)
}

func indexAll(hay, needle []byte) []int {
	var indices []int
	from := 0
	var i int
	for from < len(hay) {
		i = bytes.Index(hay[from:], needle)
		if i < 0 {
			break
		}
		indices = append(indices, from+i)
		from = from + i + 1
	}
	return indices
}

func pkgIdent(pkgpath string) string {
	li := strings.LastIndex(pkgpath, "/")
	if li == -1 {
		return pkgpath
	}
	return pkgpath[li+1:]
}

func main() {

}

/*

go oracle describe gives access to list of all things exported by a package

example:

$ oracle -format=json -pos=odb/user.go:#109 describe github.com/drathier/saiph/odb
{
        "mode": "describe",
        "describe": {
                "desc": "import of package \"github.com/drathier/saiph/odb/oauth\"",
                "pos": "C:\\gopath\\src\\github.com\\drathier\\saiph\\odb\\user.go:7:2",
                "detail": "package",
                "package": {
                        "path": "github.com/drathier/saiph/odb/oauth",
                        "members": [
                                {
                                        "name": "Provider",
                                        "type": "struct{Userid string; Token string; Secret string}",
                                        "pos": "C:\\gopath\\src\\github.com\\drathier\\saiph\\odb\\oauth\\provider.go:4:6",
                                        "kind": "type"
                                }
                        ]
                }
        }
}

By searching the source files as strings for whatever the package import statement ends with (i.e. what the package *should* be called) and checking each one of them, we can get a list of all things from that package that is used by this package. We also know where that thing is defined in that other package, so we can pull its definition out.






*/
