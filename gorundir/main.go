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
	for i := 0; i < len(nameParts)-1; i++ {
		nameParts[i] = normalize(nameParts[i])
	}
	if len(nameParts) > 0 && nameParts[0] == "" {
		nameParts = nameParts[1:]
	}

	targetFullPath := filepath.Join(cacheDir, "gorundir", strings.Join(nameParts, "-"))

	goBuild := exec.Command("go", "build", "-C", targetDir, "-o", targetFullPath, ".")
	goBuild.Stdin, goBuild.Stdout, goBuild.Stderr = nil, os.Stdout, os.Stderr
	err = goBuild.Run()
	if err != nil {
		exitErr("cannot build " + relDir)
	}

	err = syscall.Exec(targetFullPath, os.Args[1:], os.Environ())
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
