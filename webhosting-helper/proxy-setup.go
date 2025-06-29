package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates
var templatesFS embed.FS

func proxySetup(usePHP bool) {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <service entrypoint> <service webroot>\n", os.Args[0])
		os.Exit(1)
	}

	serviceEntrypoint, err := filepath.Abs(os.Args[1])
	check(err)
	serviceDir := serviceEntrypoint + ".state"
	serviceFile := filepath.Base(serviceEntrypoint)

	serviceWebroot, err := filepath.Abs(os.Args[2])
	check(err)

	os.MkdirAll(serviceDir, 0755)
	copyTemplate("templates/run", filepath.Join(serviceDir, "run"), 0755, map[string]string{
		"entrypoint_file": "'" + strings.ReplaceAll(serviceFile, `'`, `'\''`) + "'",
	})

	os.MkdirAll(serviceWebroot, 0755)
	if usePHP {
		copyTemplate("templates/htaccess", filepath.Join(serviceWebroot, ".htaccess"), 0644, nil)
		copyTemplate("templates/index.php", filepath.Join(serviceWebroot, "index.php"), 0644, map[string]string{
			"service_sock": strings.ReplaceAll(filepath.Join(serviceDir, "socket"), `'`, `\'`),
		})
	} else {
		copyTemplate("templates/htaccess2", filepath.Join(serviceWebroot, ".htaccess"), 0644, map[string]string{
			"service_sock": strings.ReplaceAll(filepath.Join(serviceDir, "socket"), `"`, `\"`),
		})
		os.RemoveAll(filepath.Join(serviceWebroot, "index.php"))
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
