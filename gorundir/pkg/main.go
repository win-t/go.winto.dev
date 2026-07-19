package gorundir

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func Main() {
	cacheDir, err := os.UserCacheDir()
	check(err)

	cacheDir = filepath.Join(cacheDir, "gorundir")
	err = os.MkdirAll(cacheDir, 0o755)
	check(err)

	ensureGo(cacheDir)

	if os.Getenv("GORUNDIR_ONLY_ENSURE_GO") != "" {
		return
	}

	if len(os.Args) < 2 {
		exitErr("gorundir: no directory is specified")
	}

	target := os.Args[1]
	targetAbs, err := filepath.Abs(target)
	check(err)

	stat, err := os.Stat(targetAbs)
	if errors.Is(err, os.ErrNotExist) || stat == nil || !stat.IsDir() {
		// TODO(win): support git:: like terraform submodules does
		exitErr("gorundir: " + target + " is not valid directory")
	}

	compiledPath := getCompiledPath(cacheDir, targetAbs)

	goBuild := exec.Command("go", "build", "-C", targetAbs, "-o", compiledPath, ".")
	goBuild.Stdin, goBuild.Stdout, goBuild.Stderr = nil, os.Stderr, os.Stderr
	err = goBuild.Run()
	if err != nil {
		exitErr("gorundir: cannot build " + target)
	}

	var args []string
	for i, arg := range os.Args[1:] {
		if i == 0 && target == "." {
			args = append(args, filepath.Base(targetAbs))
		} else {
			args = append(args, arg)
		}
	}

	err = syscall.Exec(compiledPath, args, os.Environ())
	check(err)
}
