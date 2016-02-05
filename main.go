package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/pkg/browser"
	"go/build"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
)

var (
	noTrimStructs *bool
	graphStdlib   *bool
	skipStdlib    *bool
	includeExtlibs *bool
	ranksep       *int
)

func main() {
	noTrimStructs = flag.Bool("noTrimStructs", false, "don't trim the contents of big structs to save space")
	includeExtlibs = flag.Bool("includeExtlibs", false, "include external libraries deeper than edge")
	stdlib := flag.String("graphStdlib", "edge", "graph standard library as well? yes, no or edge, where edge stops when it encounters a standard library.")

	excludeReg := flag.String("exclude", "", "exclude packages matching this regex")    // defaults to never-matching regexp
	matchReg := flag.String("match", ".*", "only include packages matching this regex") // defaults to always-matching regexp
	output := flag.String("output", "", "filename to output graphviz file to, such as graph.gv")

	ranksep = flag.Int("ranksep", 2, "distance between nodes in the graph")

	x := false
	graphStdlib = &x

	y := false
	skipStdlib = &y

	flag.Parse()
	pkg := flag.Args()

	if len(pkg) != 1 {
		fmt.Println("expected 1 argument, got", len(pkg))
		return
	}

	switch *stdlib {
	case "yes":
		*graphStdlib = true
		*skipStdlib = false
	case "edge":
		*graphStdlib = false
		*skipStdlib = false
	case "no":
		*graphStdlib = false
		*skipStdlib = true
	default:
		log.Fatalln("unknown graphStdlib value, expected 'yes', 'no' or 'edge'")
	}

	if *matchReg != "" {
		match = regexp.MustCompile(*matchReg)
	}
	if *excludeReg != "" {
		exclude = regexp.MustCompile(*excludeReg)
	}

	var err error
	cmd := exec.Command("dot", "-Tsvg")
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

	inp := io.MultiWriter(in)

	if *output != "" {
		file, err := os.Create(*output)
		if err != nil {
			panic(err)
		}
		inp = io.MultiWriter(in, file)
	}

	// stdout for now
	drawGraph(inp, pkg[0])

	in.Close()
	//drawGraph(file, "github.com/drathier/saiph/grammar/parser")

	go browser.OpenReader(out)

	cmd.Wait()
}

var (
	match   *regexp.Regexp
	exclude *regexp.Regexp
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
		file, err := ioutil.ReadFile(absPath(filename, ourPkg.ImportPath))
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
				oracle, err := oracleDescribe(pos, filename, ourPath)
				if err != nil {
					// `ambiguous selection within source file` -> comment
					// `no identifier here` -> import statement
					return // false positive; shadowed variable, string or comment
				}

				oracleDef, err := oracleDefine(pos + len(pkgName) + 1, filename, ourPath)
				if err != nil {
					return // false positive; probably an import statement
				}

				if strings.HasPrefix(oracleDef.Definition.Desc, "var") {
					return // variable definition; breaks code; TODO: handle this as well.
				}

				// 4. add things to result
				d := dep{from: ourPath, to: oracle.Describe.Pkg.Path, typ: oracleDef.Definition.Desc}

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
		indices = append(indices, from + i)
		from = from + i + 1
	}
	return indices
}

func pkgIdent(pkgpath string) string {
	li := strings.LastIndex(pkgpath, "/")
	if li == -1 {
		return pkgpath
	}
	return pkgpath[li + 1:]
}

func drawGraph(in io.Writer, path string) {
	depmap := make(map[string][]string) // depends on

	fmt.Fprintf(in, "digraph %q {\n", path)
	fmt.Fprintf(in, "\tgraph [ranksep=%d];\n", *ranksep)
	fmt.Fprintf(in, "\tcompound=true;\n")
	fmt.Fprintf(in, "\tsubgraph %q {\n", "cluster_" + path)
	fmt.Fprintf(in, "\t\tlabel=%q;\n", path)
	fmt.Fprintf(in, "\t\t%q [shape=point, style=invis];\n", path)
	fmt.Fprintf(in, "\t}\n")

	// list of types inside a package that is used by anyone
	used := make(map[string][]string)
	seen := make(map[dep]struct{})

	fmt.Fprintf(in, "\n\t// dependencies to types, variables etc.\n")
	importPkg(depmap, path)
	for from, tos := range depmap {
		if !*includeExtlibs && !isStdlib(from) && !strings.HasPrefix(from, path) {
			log.Println("skipping extlib", from, path)
			continue
		}
		for _, to := range tos {
			fmt.Println(from, "->", to)
			//if strings.HasPrefix(from, "github.com") && strings.HasPrefix(to, "github.com") && !strings.Contains(from, "couchbase") {
			switch {
			case !*graphStdlib && isStdlib(from): // only plot edge
				continue
			case !match.MatchString(to):
				continue
			case isStdlib(to) && *skipStdlib: // don't plot any stdlib
				continue
			case exclude != nil && exclude.MatchString(from):
				continue
			}
			log.Println("ok", from, "->", to)
			// TODO Add filter so pkgs shown can be filtered by regex.
			nextDep:
			for _, obj := range findDeps(from, to) {
				fmt.Println(obj)

				if _, found := seen[obj]; found {
					continue nextDep
				}
				seen[obj] = struct{}{}

				fmt.Fprintf(in, "\t%q -> %q [color=%q, ltail=%q];\n", obj.from, obj.typ, color(obj.from), "cluster_" + obj.from)
				used[obj.to] = append(used[obj.to], obj.typ)
			}
			if _, found := used[from]; !found {
				used[from] = []string{}
			}
		}
	}

	fmt.Fprintf(in, "\n\t// edges for empty imports\n")
	// add dependency edges for packages that include other packages, but don't use anything in them, i.e. underscore imports
	for from, tos := range depmap {
		if !*includeExtlibs && !isStdlib(from) && !strings.HasPrefix(from, path) {
			log.Println("skipping extlib", from, path)
			continue
		}
		if *graphStdlib || !isStdlib(from) {
			for _, to := range tos {
				switch {
				case !*graphStdlib && isStdlib(from): // only plot edge
					continue
				case !match.MatchString(to):
					continue
				case isStdlib(to) && *skipStdlib: // don't plot any stdlib
					continue
				case exclude != nil && exclude.MatchString(from):
					continue
				}
				if _, found := used[from]; !found {
					fmt.Fprintf(in, "\t%q -> %q [color=%q, ltail=%q, lhead=%q, style=dashed];\n", from, to, color(from), "cluster_" + from, "cluster_" + to)
				}
			}
		}
	}

	fmt.Fprintf(in, "\n\t// subgraphs\n")
	// subgraphs
	for pkg, types := range used {
		if pkg == "" {
			pkg = "unknown package(s)"
		}
		fmt.Fprintf(in, "\tsubgraph %q {\n", "cluster_" + pkg)
		fmt.Fprintf(in, "\t\tlabel=%q;\n", pkg)
		fmt.Fprintf(in, "\t\tcolor=%q;\n", color(pkg))
		fmt.Fprintf(in, "\t\t%q [weight=0, shape=point, style=invis];\n", pkg)
		for _, t := range types {
			// clear out redundant path
			// last dot in type marks delim; remove before that

			fmt.Println("stripping", t)
			label := t
			mid := strings.Index(label, " ") + 1
			fmt.Println("mid", mid)
			endmid := mid + strings.IndexAny(label[mid + 1:], " (") + 1
			fmt.Println("endmid", endmid)

			if !*noTrimStructs {
				if strings.HasPrefix(strings.TrimSpace(label[endmid:]), "struct{") {
					label = label[:endmid] + " struct{...}"
				}
			}
			fmt.Println("trimmed struct", t, "->", label)

			li := strings.LastIndex(label[mid:endmid], ".")
			if li >= 0 {
				fmt.Println("lastindex", li)
				label = label[:mid] + label[mid + li + 1:]
			}

			// remove pkg path, if it's in the same pkg
			label = strings.Replace(label, t[mid:mid + li + 1], "", -1)

			fmt.Println("stripped", t, "->", label)

			fmt.Fprintf(in, "\t\t%q [weight=1, label=%q];\n", t, label)
		}
		fmt.Fprintf(in, "\t}\n")
	}

	fmt.Fprintf(in, "}\n")
	fmt.Printf("depmap: %#v\n", depmap)
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
