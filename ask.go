package main

import (
	"fmt"
	"go/build"
	"math/rand"
	"sync"
)

func isStdlib(path string) bool {
	p, err := build.Import(path, "", 0)
	if err != nil {
		panic(err)
	}
	return p.Goroot
}

var colors = make(map[string][3]int)
var colorlock sync.Mutex

func color(path string) string {
	colorlock.Lock()
	defer colorlock.Unlock()

	if _, found := colors[path]; !found {
		colors[path] = [3]int{rand.Intn(192), rand.Intn(192), rand.Intn(192)}
	}
	return fmt.Sprintf("#%02x%02x%02x", colors[path][0], colors[path][1], colors[path][2])
}
