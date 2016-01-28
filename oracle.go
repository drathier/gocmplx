package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

type oracle struct {
	Mode     string
	Describe odesc
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

func oracleLookup(pos int, file string, pkg string) (oracle, error) {
	fmt.Printf("oracleLookup(pos %d, file %s, pkg %s)\n", pos, file, pkg)
	// call the oracle
	cmd := exec.Command("oracle", "-format=json", fmt.Sprintf("-pos=%s:#%d", absPath(file, pkg), pos), "describe", pkg)

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
