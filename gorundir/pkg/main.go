package pkg

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"time"
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

	relDir := os.Args[1]
	targetDir, err := filepath.Abs(relDir)
	check(err)

	stat, err := os.Stat(targetDir)
	if errors.Is(err, os.ErrNotExist) || stat == nil || !stat.IsDir() {
		exitErr("gorundir: " + relDir + " is not valid directory")
	}

	nameParts := strings.Split(targetDir, string(os.PathSeparator))
	for i := range nameParts {
		nameParts[i] = normalize(nameParts[i])
	}
	if len(nameParts) > 0 && nameParts[0] == "" {
		nameParts = nameParts[1:]
	}

	targeDirSum := sha256.Sum256([]byte(targetDir))
	compiledPath := filepath.Join(cacheDir, strings.Join(nameParts, "-")) + "-" + hex.EncodeToString(targeDirSum[:])[:8]

	goBuild := exec.Command("go", "build", "-C", targetDir, "-o", compiledPath, ".")
	goBuild.Stdin, goBuild.Stdout, goBuild.Stderr = nil, os.Stderr, os.Stderr
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
	panic("os.Exit returned")
}

func httpGet(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(req)
}

func ensureGo(cacheDir string) {
	if _, err := exec.LookPath("go"); err == nil {
		return // use existing go binary
	}

	ourGoPath := filepath.Join(cacheDir, "go", "bin")

	err := os.Setenv("PATH", ourGoPath+string(os.PathListSeparator)+os.Getenv("PATH"))
	check(err)

	if _, err := os.Stat(filepath.Join(ourGoPath, "go")); err == nil {
		return // already downloaded before
	}

	if !slices.Contains([]string{"linux", "darwin"}, runtime.GOOS) {
		// TODO(win): also support auto download for other OS, currently i don't have a way to test it
		exitErr("gorundir: Please install Go")
	}

	downloadDir := filepath.Join(cacheDir, "go-download")
	err = os.MkdirAll(downloadDir, 0o755)
	check(err)

	lockFile, err := os.OpenFile(filepath.Join(downloadDir, "lock"), os.O_CREATE|os.O_RDWR, 0o644)
	check(err)
	defer lockFile.Close()

	for until := time.Now().Add(5 * time.Minute); true; time.Sleep(2 * time.Second) {
		err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			break
		}
		if time.Now().After(until) {
			exitErr("gorundir: Please install Go, auto-install is failing when acquiring lock")
		}
	}

	if _, err := os.Stat(filepath.Join(ourGoPath, "go")); err == nil {
		return
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	resp, err := httpGet(ctx, "https://go.dev/VERSION?m=text")
	check(err)
	respBody, err := io.ReadAll(resp.Body)
	check(err)
	resp.Body.Close()

	goVersion := strings.Split(string(respBody), "\n")[0]
	resp, err = httpGet(ctx, "https://go.dev/dl/"+goVersion+"."+runtime.GOOS+"-"+runtime.GOARCH+".tar.gz")
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
			continue // TODO(win): check if this enough?, no symlink or other exotic things in the tar file
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
