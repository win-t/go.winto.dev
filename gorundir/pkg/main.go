package gorundir

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	var abs, bin string
	if strings.HasPrefix(target, "git::https://") { // TODO(win): support non git::https:// case
		abs, bin = computeGitPath(cacheDir, target)
	} else {
		abs, bin = computeLocalPath(cacheDir, target)
	}

	if output, err := exec.Command("go", "build", "-C", abs, "-o", bin, ".").CombinedOutput(); err != nil {
		os.Stderr.Write(output)
		os.Stderr.WriteString("\n")
		exitErr("gorundir: cannot build " + target)
	}

	var args []string
	for i, arg := range os.Args[1:] {
		if i == 0 && target == "." {
			args = append(args, filepath.Base(abs))
		} else {
			args = append(args, arg)
		}
	}

	err = syscall.Exec(bin, args, os.Environ())
	check(err)
}
