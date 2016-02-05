package util

import "os"

func AbsPath(filename, pkg string) string {
	return os.Getenv("GOPATH") + "/src/" + pkg + "/" + filename
}
