package plot

import (
	"fmt"
	"io"
	"strings"
)

func DrawGraph(in io.Writer, path string) {
	depmap := make(map[string][]string) // depends on

	fmt.Fprintf(in, "digraph %q {\n", path)
	fmt.Fprintf(in, "\tgraph [ranksep=%d];\n", *ranksep)
	fmt.Fprintf(in, "\tcompound=true;\n")
	fmt.Fprintf(in, "\tsubgraph %q {\n", "cluster_"+path)
	fmt.Fprintf(in, "\t\tlabel=%q;\n", path)
	fmt.Fprintf(in, "\t\t%q [shape=point, style=invis];\n", path)
	fmt.Fprintf(in, "\t}\n")

	// list of types inside a package that is used by anyone
	used := make(map[string][]string)
	seen := make(map[dep]struct{})

	fmt.Fprintf(in, "\n\t// dependencies to types, variables etc.\n")
	importPkg(depmap, path)
	for from, tos := range depmap {
		for _, to := range tos {
			fmt.Println(from, "->", to)
			//if strings.HasPrefix(from, "github.com") && strings.HasPrefix(to, "github.com") && !strings.Contains(from, "couchbase") {
			if graphStdlibEdges.Whole() || !isStdlib(from) {
				if !match.MatchString(from) || (exclude != nil && exclude.MatchString(from)) {
					continue
				}
				// TODO Add filter so pkgs shown can be filtered by regex.

			nextDep:
				for _, obj := range findDeps(from, to) {
					fmt.Println(obj)

					if _, found := seen[obj]; found {
						continue nextDep
					}
					seen[obj] = struct{}{}

					fmt.Fprintf(in, "\t%q -> %q [color=%q, ltail=%q];\n", obj.from, obj.typ, color(obj.from), "cluster_"+obj.from)
					used[obj.to] = append(used[obj.to], obj.typ)
				}
				if _, found := used[from]; !found {
					used[from] = []string{}
				}
			}
		}
	}

	fmt.Fprintf(in, "\n\t// edges for empty imports\n")
	// add dependency edges for packages that include other packages, but don't use anything in them, i.e. underscore imports
	for from, tos := range depmap {
		if graphStdlibEdges.Whole() || !isStdlib(from) {
			if !match.MatchString(from) || (exclude != nil && exclude.MatchString(from)) {
				continue
			}
			for _, to := range tos {
				if _, found := used[from]; !found {
					fmt.Fprintf(in, "\t%q -> %q [color=%q, ltail=%q, lhead=%q, style=dashed];\n", from, to, color(from), "cluster_"+from, "cluster_"+to)
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
		fmt.Fprintf(in, "\tsubgraph %q {\n", "cluster_"+pkg)
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
			endmid := mid + strings.IndexAny(label[mid+1:], " (") + 1
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
				label = label[:mid] + label[mid+li+1:]
			}

			// remove pkg path, if it's in the same pkg
			label = strings.Replace(label, t[mid:mid+li+1], "", -1)

			fmt.Println("stripped", t, "->", label)

			fmt.Fprintf(in, "\t\t%q [weight=1, label=%q];\n", t, label)
		}
		fmt.Fprintf(in, "\t}\n")
	}

	fmt.Fprintf(in, "}\n")
	fmt.Printf("depmap: %#v\n", depmap)
}
