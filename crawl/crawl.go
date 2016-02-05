// Package crawl traverses the package directory and fetches the packages needed, and parses them.
package crawl

import (
	"bytes"
	"fmt"
	"github.com/drathier/gocmplx/oracle"
	"github.com/drathier/gocmplx/util"
	"go/build"
	"io/ioutil"
	"strings"
	"sync"
)

type Task struct {
	path        string
	graphStdlib string // yes, edge or no
}

func (t Task) GraphStdlibEdge() bool {
	return t.graphStdlib == "yes" || t.graphStdlib == "edge"
}
func (t Task) GraphStdlibDeep() bool {
	return t.graphStdlib == "yes"
}

func Crawl(t Task) map[string][]string {
	depmap := make(map[string][]string)
	importPkg(depmap, t.path, t, false)
	return depmap
}

// importPkg recursively traverses the whole import DAG and returns a map containing the list of edges each edge imports directly
func importPkg(depmap map[string][]string, path string, t Task, skipGoroot bool) {
	if _, found := depmap[t.path]; found {
		return // already imported this
	}
	pkgs, err := build.Import(path, "", 0)
	if err != nil {
		panic(err)
	}

	if pkgs.Goroot {
		switch {
		case skipGoroot:
			return
		case !t.GraphStdlibEdge() && len(depmap) > 0: // don't skip first package, even if it's in Goroot
			return
		case !t.GraphStdlibDeep():
			skipGoroot = true
		}
	}
	for _, pkg := range pkgs.Imports {
		depmap[path] = append(depmap[path], pkg)
		importPkg(depmap, pkg, t, skipGoroot)
	}
}

type dep struct {
	from string // current file path
	to   string // oracle describe/package/path
	typ  string // oracle describe/package/members[i]/type
}

// saiph.User depends on gocmplx.Deps; return filename.go:line:col imports filename2.go:line:col type struct{asd string; potato int}
// path = the package we are in; pkgpath = the package we imported
func findDeps(ourPath, otherPkg string) []dep {
	// 1. get package name from pkgPath
	pkgName := pkgIdent(otherPkg)
	fmt.Println("pkgName", pkgName)

	// 2. find that package name in source files
	ourPkg, err := build.Import(ourPath, "", 0)
	if err != nil {
		panic(err)
	}

	var depm = make(map[dep]struct{})
	var depmlock sync.Mutex
	var depmsema = make(chan struct{}, 64)
	var wg sync.WaitGroup

	// check all our source files for references to otherPkg
	for _, filename := range ourPkg.GoFiles {
		fmt.Println("pkg.GoFiles", filename)
		file, err := ioutil.ReadFile(util.AbsPath(filename, ourPkg.ImportPath))
		if err != nil {
			panic(err)
		}
		for _, pos := range indexAll(file, []byte(pkgName)) {
			wg.Add(1)
			depmsema <- struct{}{}
			go func(pos int, filename string, outPath string) {
				defer func() {
					wg.Done()
					<-depmsema
				}()

				// 3. check if those matches are in fact pointing to the package
				o, err := oracle.Describe(pos, filename, ourPath)
				if err != nil {
					// `ambiguous selection within source file` -> comment
					// `no identifier here` -> import statement
					return // false positive; shadowed variable, string or comment
				}

				oracleDef, err := oracle.Define(pos+len(pkgName)+1, filename, ourPath)
				if err != nil {
					return // false positive; probably an import statement
				}

				if strings.HasPrefix(oracleDef.Definition.Desc, "var") {
					return // variable definition; breaks code; TODO: handle this as well.
				}

				// 4. add things to result
				d := dep{from: ourPath, to: o.Describe.Pkg.Path, typ: oracleDef.Definition.Desc}

				depmlock.Lock()
				depm[d] = struct{}{}
				depmlock.Unlock()
				fmt.Println("pkg points to dep here, using valid reference; store it for later use")
			}(pos, filename, ourPath)
		}
	}
	wg.Wait()
	var deps []dep
	for d := range depm {
		deps = append(deps, d)
	}
	return deps
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

/*

FIXME:
missing label on anaconda; imported by others, but does not import anything itself

*/
