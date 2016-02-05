package oracle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/drathier/gocmplx/util"
	"os/exec"
	"sync"
)

type oracle struct {
	Mode       string
	Describe   odesc
	Definition odef
}

type odef struct {
	Objpos string
	Desc   string
}

type odesc struct {
	Desc   string
	Pos    string
	Detail string
	Pkg    opkg `json:"package"`
}

type opkg struct {
	Path    string
	Members []omember
}

type omember struct {
	Name string
	Typ  string `json:"type"`
	Pos  string
	Kind string
}

func Describe(pos int, file string, pkg string) (oracle, error) {
	return gen(pos, file, pkg, "describe")
}

func Define(pos int, file string, pkg string) (oracle, error) {
	return gen(pos, file, pkg, "definition")
}

type oracledp struct {
	pos  int
	file string
	pkg  string
	op   string
}

type oracledpout struct {
	o   oracle
	err error
}

var mem = make(map[oracledp]oracledpout)
var memlock = make(map[oracledp]*sync.Once)
var lock sync.Mutex

// gen is a thread-safe memoization wrapper around genImpl
func gen(pos int, file string, pkg string, op string) (oracle, error) {
	key := oracledp{
		pos:  pos,
		file: file,
		pkg:  pkg,
		op:   op,
	}

	lock.Lock()
	if _, found := memlock[key]; !found {
		memlock[key] = new(sync.Once)
	}
	once := memlock[key]
	lock.Unlock()

	once.Do(func() {
		oracle, err := genImpl(key.pos, key.file, key.pkg, key.op)
		res := oracledpout{
			o:   oracle,
			err: err,
		}
		lock.Lock()
		mem[key] = res
		lock.Unlock()
	})

	lock.Lock()
	ans := mem[key]
	lock.Unlock()
	return ans.o, ans.err
}

// genImpl executes the calls to oracle
func genImpl(pos int, file string, pkg string, op string) (oracle, error) {
	fmt.Printf("oracle_%s(pos %d, file %s, pkg %s)\n", op, pos, file, pkg)
	// call the oracle
	cmd := exec.Command("oracle", "-format=json", fmt.Sprintf("-pos=%s:#%d", util.AbsPath(file, pkg), pos), op, pkg)

	var out bytes.Buffer
	cmd.Stdout = &out
	var errout bytes.Buffer
	cmd.Stderr = &errout
	err := cmd.Run()
	fmt.Println("stderr:", errout.String())
	var o oracle
	if err != nil {
		return o, err
	}

	fmt.Println("run ora")

	err = json.Unmarshal(out.Bytes(), &o)
	fmt.Println("run marshal")
	if err != nil {
		return o, err
	}

	fmt.Printf("oracle output: %#v\n", o)
	return o, nil
}
