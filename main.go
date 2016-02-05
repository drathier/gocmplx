package main

import (
	"flag"
	"fmt"
	"github.com/drathier/gocmplx/plot"
	"github.com/pkg/browser"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
)

type graphStdlibSetting byte

func (g graphStdlibSetting) Nothing() bool {
	return g == 0
}
func (g graphStdlibSetting) Edges() bool {
	return (g & graphStdlibEdge) != 0
}
func (g graphStdlibSetting) Whole() bool {
	return (g & graphWholeStdlib) != 0
}

const (
	GraphNoStdlib    = 0
	graphStdlibEdge  = 1
	graphWholeStdlib = graphStdlibEdge | 2
)

var (
	noTrimStructs    *bool
	graphStdlibEdges graphStdlibSetting
	skipStdlib       *bool
	ranksep          *int
)

func main() {
	noTrimStructs = flag.Bool("noTrimStructs", false, "don't trim the contents of big structs to save space")
	stdlib := flag.String("graphStdlib", "edge", "graph standard library as well? yes, no or edge, where edge stops when it encounters a standard library.")

	excludeReg := flag.String("exclude", "", "exclude packages matching this regex") // defaults to never-matching regexp
	output := flag.String("output", "", "filename to output graphviz file to, such as graph.gv")

	ranksep = flag.Int("ranksep", 2, "distance between nodes in the graph; increase for big graphs, to make them more readable")

	flag.Parse()
	pkg := flag.Args()

	if len(pkg) != 1 {
		fmt.Println("expected 1 argument, got", len(pkg))
		return
	}

	switch *stdlib {
	case "yes":
		graphStdlibEdges = graphWholeStdlib
	case "edge":
		graphStdlibEdges = graphStdlibEdge
	case "no":
		graphStdlibEdges = GraphNoStdlib
	default:
		log.Fatalln("unknown graphStdlib value, expected 'yes', 'no' or 'edge'")
	}

	match = regexp.MustCompile(pkg[0])
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
	plot.DrawGraph(inp, pkg[0])

	in.Close()
	//drawGraph(file, "github.com/drathier/saiph/grammar/parser")

	go browser.OpenReader(out)

	cmd.Wait()
}

var (
	match   *regexp.Regexp
	exclude *regexp.Regexp
)
