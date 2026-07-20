package gorundir

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func exitErr(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
	panic("os.Exit returned")
}

var nonAlphaNum = regexp.MustCompile("[^a-zA-Z0-9]+")

func normalizePart(name string) string {
	name = nonAlphaNum.ReplaceAllString(name, "")
	if len(name) > 6 {
		name = name[:6]
	}
	return name
}

func normalize(targetAbsDir string) string {
	nameParts := strings.Split(targetAbsDir, string(os.PathSeparator))
	for i := range nameParts {
		nameParts[i] = normalizePart(nameParts[i])
	}
	if len(nameParts) > 0 && nameParts[0] == "" {
		nameParts = nameParts[1:]
	}

	targeDirSum := sha256.Sum256([]byte(targetAbsDir))
	return strings.Join(nameParts, "-") + "-" + hex.EncodeToString(targeDirSum[:])[:8]
}

func computeLocalPath(cacheDir, target string) (abs, bin string) {
	var err error
	abs, err = filepath.Abs(target)
	check(err)

	stat, err := os.Stat(abs)
	if errors.Is(err, os.ErrNotExist) || stat == nil || !stat.IsDir() {
		exitErr("gorundir: " + target + " is not valid directory")
	}

	bin = filepath.Join(cacheDir, "bin", normalize(abs))

	return abs, bin
}
