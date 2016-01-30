package main

import (
	"bytes"
	"fmt"
	"go/build"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

func importPkg(depmap map[string][]string, path string) {
	if _, found := depmap[path]; found {
		return // already imported this
	}
	pkgs, err := build.Import(path, "", 0)
	if err != nil {
		panic(err)
	}
	for _, pkg := range pkgs.Imports {
		depmap[path] = append(depmap[path], pkg)
		importPkg(depmap, pkg)
	}
}

type dep struct {
	from string // current file path
	to   string // oracle describe/package/path
	typ  string // oracle describe/package/members[i]/type
	//toIsStdlib bool   // is "to" a stdlib? FIXME: implement this; ask the oracle or something
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
	// check all our source files for references to otherPkg
	for _, filename := range ourPkg.GoFiles {
		fmt.Println("pkg.GoFiles", filename)
		file, err := ioutil.ReadFile(absPath(filename, ourPkg.ImportPath))
		if err != nil {
			panic(err)
		}
		for _, pos := range indexAll(file, []byte(pkgName)) {
			// 3. check if those matches are in fact pointing to the package
			oracle, err := oracleDescribe(pos, filename, ourPath)
			if err != nil {
				// `ambiguous selection within source file` -> comment
				// `no identifier here` -> import statement
				continue // false positive; shadowed variable, string or comment
			}


			oracleDef, err := oracleDefine(pos+len(pkgName)+1, filename, ourPath)
			if err != nil {
				continue // false positive; probably an import statement
			}

			if strings.HasPrefix(oracleDef.Definition.Desc, "var") {
				continue // variable definition; breaks code; TODO: handle this as well.
			}

			// 4. add things to result
			d := dep{from: ourPath, to: oracle.Describe.Pkg.Path, typ: oracleDef.Definition.Desc}
			depm[d] = struct{}{}
			fmt.Println("pkg points to dep here, using valid reference; store it for later use")
		}
	}
	var deps []dep
	for d := range depm {
		deps = append(deps, d)
	}
	return deps
}

func absPath(filename, pkg string) string {
	return os.Getenv("GOPATH") + "/src/" + pkg + "/" + filename
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

func drawGraph(in io.WriteCloser, path string) {
	depmap := make(map[string][]string) // depends on

	fmt.Fprintf(in, "digraph {\n")
	fmt.Fprintf(in, "\tcompound=true;\n")
	fmt.Fprintf(in, "\tsubgraph %q {\n", "cluster_"+path)
	fmt.Fprintf(in, "\t\t label=%q;\n", path)
	fmt.Fprintf(in, "\t\t%q [shape=point, style=invis];\n", path)
	fmt.Fprintf(in, "\t}\n")

	for from := range depmap {
		fmt.Fprintf(in, "\t%q [shape=box];\n", from)
	}

	// list of types inside a package that is used by anyone
	used := make(map[string][]string)

	importPkg(depmap, path)
	for from, tos := range depmap {
		for _, to := range tos {
			fmt.Println(from, "->", to)
			if strings.HasPrefix(from, "github.com") && strings.HasPrefix(to, "github.com") && !strings.Contains(from, "couchbase") {
				//fmt.Fprintf(in, "\t%q -> %q [weight=1];\n", from, to)
				//continue
				// fucking ugly hack; oracle has info if this is a stdlib or not; use that instead. Also add filter so pkgs shown can be filtered by regex.
				for _, obj := range findDeps(from, to) {
					fmt.Println(obj)
					fmt.Fprintf(in, "\t%q -> %q [ltail=%q];\n", obj.from, obj.typ, "cluster_"+obj.from)
					used[obj.to] = append(used[obj.to], obj.typ)
				}
			}
		}
	}

	// subgraphs
	for pkg, types := range used {
		fmt.Fprintf(in, "\tsubgraph %q {\n", "cluster_"+pkg)
		fmt.Fprintf(in, "\t\tlabel=%q;\n", pkg)
		for _, t := range types {
			fmt.Fprintf(in, "\t\t%q [weight=1];\n", t)
		}
		fmt.Fprintf(in, "\t}\n")
	}

	fmt.Fprintf(in, "}\n")
	in.Close()
	fmt.Printf("%#v\n", depmap)
}

func main() {
	var err error
	/*cmd := exec.Command("dot", "-Tsvg")
	in, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}
	out, err := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		panic(err)
	}
	*/
	file, err := os.Create("output.gv")
	if err != nil {
		panic(err)
	}

	// stdout for now
	drawGraph(file, "github.com/drathier/saiph/odb")

	//go browser.OpenReader(out)

	//cmd.Wait()
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
