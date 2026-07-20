//go:build linux || darwin

package gorundir

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

func ensureGo(cacheDir string) {
	if _, err := exec.LookPath("go"); err == nil {
		return // use existing go binary
	}

	ourGoPath := filepath.Join(cacheDir, "go", "bin")

	err := os.Setenv("PATH", ourGoPath+string(os.PathListSeparator)+os.Getenv("PATH"))
	check(err)

	// already downloaded before?
	if _, err := os.Stat(filepath.Join(ourGoPath, "go")); err == nil {
		// TODO(win): auto upgrade if newer version available
		return
	}

	downloadDir := filepath.Join(cacheDir, "go-download")
	err = os.MkdirAll(downloadDir, 0o755)
	check(err)

	unlock := getLock(filepath.Join(downloadDir, "lock"), "gorundir: Please install Go, auto-install is failing when acquiring lock")
	defer unlock()

	if _, err := os.Stat(filepath.Join(ourGoPath, "go")); err == nil {
		return
	}

	resp, err := http.Get("https://go.dev/VERSION?m=text")
	check(err)
	respBody, err := io.ReadAll(resp.Body)
	check(err)
	resp.Body.Close()

	goVersion := strings.Split(string(respBody), "\n")[0]
	resp, err = http.Get("https://go.dev/dl/" + goVersion + "." + runtime.GOOS + "-" + runtime.GOARCH + ".tar.gz")
	check(err)
	defer resp.Body.Close()

	gzipReader, err := gzip.NewReader(resp.Body)
	check(err)
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		check(err)
		if header.Typeflag != tar.TypeReg {
			continue // TODO(win): no symlink or other exotic things in the tar file?
		}
		targetPath := filepath.Join(downloadDir, header.Name)

		err = os.MkdirAll(filepath.Dir(targetPath), 0o755)
		check(err)

		f, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
		check(err)

		_, err = io.Copy(f, tarReader)
		check(err)

		f.Close()
	}

	err = os.Rename(filepath.Join(downloadDir, "go"), filepath.Join(cacheDir, "go"))
	check(err)
}

func getLock(lockPath string, unlockErr string) func() {
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	check(err)

	for until := time.Now().Add(5 * time.Minute); true; time.Sleep(2 * time.Second) {
		err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			break
		}
		if time.Now().After(until) {
			exitErr(unlockErr)
		}
	}

	lockFile.Truncate(0)
	fmt.Fprintln(lockFile, os.Getpid())

	return func() { lockFile.Truncate(0); lockFile.Close() }
}
