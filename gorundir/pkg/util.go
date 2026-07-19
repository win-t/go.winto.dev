package gorundir

import (
	"crypto/sha256"
	"encoding/hex"
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

func normalize(name string) string {
	name = nonAlphaNum.ReplaceAllString(name, "")
	if len(name) > 6 {
		name = name[:4] + ".."
	}
	return name
}

func getCompiledPath(cacheDir string, targetAbsDir string) string {
	nameParts := strings.Split(targetAbsDir, string(os.PathSeparator))
	for i := range nameParts {
		nameParts[i] = normalize(nameParts[i])
	}
	if len(nameParts) > 0 && nameParts[0] == "" {
		nameParts = nameParts[1:]
	}

	targeDirSum := sha256.Sum256([]byte(targetAbsDir))
	return filepath.Join(cacheDir, strings.Join(nameParts, "-")) + "-" + hex.EncodeToString(targeDirSum[:])[:8]
}
