package main

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates
var templatesFS embed.FS

func proxySetup() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <service dir> <service webroot>\n", os.Args[0])
		os.Exit(1)
	}

	serviceDir, err := filepath.Abs(os.Args[1])
	check(err)
	serviceWebroot, err := filepath.Abs(os.Args[2])
	check(err)

	os.MkdirAll(serviceDir, 0755)
	if f := filepath.Join(serviceDir, "run"); fileNotExists(f) {
		copyTemplate("templates/run", f, 0755, nil)
	}
	if f := filepath.Join(serviceDir, "app"); fileNotExists(f) {
		copyTemplate("templates/app", f, 0755, nil)
	}

	os.MkdirAll(serviceWebroot, 0755)
	if f := filepath.Join(serviceWebroot, ".htaccess"); fileNotExists(f) {
		copyTemplate("templates/htaccess", f, 0644, nil)
	}
	if f := filepath.Join(serviceWebroot, "index.php"); fileNotExists(f) {
		copyTemplate("templates/index.php", f, 0644, map[string]string{
			"service_sock": "'" + strings.ReplaceAll(filepath.Join(serviceDir, "socket"), `'`, `\'`) + "'",
		})
	}
}

func copyTemplate(srcPath, dstPath string, mode os.FileMode, v any) {
	tmpl, err := template.ParseFS(templatesFS, srcPath)
	check(err)

	dst, err := os.Create(dstPath)
	check(err)
	defer dst.Close()

	err = tmpl.Execute(dst, v)
	check(err)

	err = os.Chmod(dstPath, mode)
	check(err)
}

func fileNotExists(path string) bool {
	_, err := os.Stat(path)
	return errors.Is(err, os.ErrNotExist)
}
