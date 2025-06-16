package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

func main() {
	if len(os.Args) < 2 {
		exitErr("no directory is specified")
	}

	cacheDir, err := os.UserCacheDir()
	check(err)

	relDir := os.Args[1]
	targetDir, err := filepath.Abs(relDir)
	check(err)

	stat, err := os.Stat(targetDir)
	if errors.Is(err, os.ErrNotExist) || stat == nil || !stat.IsDir() {
		exitErr(relDir + " is not valid directory")
	}

	nameParts := strings.Split(targetDir, string(os.PathSeparator))
	for i := range nameParts {
		nameParts[i] = normalize(nameParts[i])
	}
	if len(nameParts) > 0 && nameParts[0] == "" {
		nameParts = nameParts[1:]
	}

	compiledPath := filepath.Join(cacheDir, "gorundir", strings.Join(nameParts, "-"))

	goBuild := exec.Command("go", "build", "-C", targetDir, "-o", compiledPath, ".")
	goBuild.Stdin, goBuild.Stdout, goBuild.Stderr = nil, os.Stdout, os.Stderr
	err = goBuild.Run()
	if err != nil {
		exitErr("gorundir: cannot build " + relDir)
	}

	var args []string
	for i, arg := range os.Args[1:] {
		if i == 0 && relDir == "." {
			args = append(args, filepath.Base(targetDir))
		} else {
			args = append(args, arg)
		}
	}

	err = syscall.Exec(compiledPath, args, os.Environ())
	check(err)
}

var nonAlphaNum = regexp.MustCompile("[^a-zA-Z0-9]+")

func normalize(name string) string {
	name = nonAlphaNum.ReplaceAllString(name, "")
	if len(name) > 6 {
		name = name[:4] + ".."
	}
	return name
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func exitErr(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
