package pkg

import (
	"archive/tar"
	"compress/gzip"
	"context"
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
	"strings"
	"syscall"
	"time"
)

func Main() {
	cacheDir, err := os.UserCacheDir()
	check(err)

	cacheDir = filepath.Join(cacheDir, "gorundir")
	os.MkdirAll(cacheDir, 0755)

	downloadAndInstallGo(cacheDir)

	if os.Getenv("GORUNDIR_ONLY_ENSURE_GO") != "" {
		return
	}

	if len(os.Args) < 2 {
		exitErr("no directory is specified")
	}

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

	compiledPath := filepath.Join(cacheDir, strings.Join(nameParts, "-"))

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
}

func httpGet(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(req)
}

func downloadAndInstallGo(cacheDir string) {
	if _, err := exec.LookPath("go"); err == nil {
		return
	}

	os.Setenv("PATH", filepath.Join(cacheDir, "go", "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
	if _, err := exec.LookPath("go"); err == nil {
		return
	}

	if runtime.GOOS == "windows" {
		exitErr("gorundir: Please install Go") // TODO(win): also support auto download for Windows
	}

	downloadDir := filepath.Join(cacheDir, "go-download")
	if err := os.Mkdir(downloadDir, 0755); errors.Is(err, os.ErrExist) {
		// If the directory already exists, we assume that the download is in progress.
		for until := time.Now().Add(5 * time.Minute); time.Now().Before(until); {
			if _, err := os.Stat(downloadDir); errors.Is(err, os.ErrNotExist) {
				break
			}
			time.Sleep(1 * time.Second)
		}
		return
	}
	defer os.RemoveAll(downloadDir)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	resp, err := httpGet(ctx, "https://go.dev/VERSION?m=text")
	check(err)
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	check(err)

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
			continue
		}
		func() {
			targetPath := filepath.Join(downloadDir, header.Name)
			os.MkdirAll(filepath.Dir(targetPath), 0755)
			f, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			check(err)
			defer f.Close()
			_, err = io.Copy(f, tarReader)
			check(err)
		}()
	}

	err = os.Rename(filepath.Join(downloadDir, "go"), filepath.Join(cacheDir, "go"))
	check(err)
}
