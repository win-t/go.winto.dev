package gorundir

import (
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

// parseGitSource takes a git:: module source (with the git:: prefix) and
// returns the repo URL, the git ref, and the subdir.
//
// Splitting mirrors Terraform's SplitPackageSubdir: the //subdir is separated
// while ignoring the scheme's "://", any trailing ?query on the subdir is moved
// back onto the repo URL, and ?ref= is then extracted. Remaining query params
// (e.g. depth, sshkey) stay on the returned repo URL.
//
// AI-Generated code
func parseGitSource(src string) (repoURL, ref, subdir string) {
	src = strings.TrimPrefix(src, "git::")

	pkg, subdir := splitSubdir(src)

	u, err := url.Parse(pkg)
	if err != nil {
		exitErr("gorundir: " + src + " is not valid remote git format")
	}
	q := u.Query()
	ref = q.Get("ref")
	u.RawQuery = ""

	return u.String(), ref, subdir
}

// splitSubdir is a faithful port of Terraform's SplitPackageSubdir.
//
// AI-Generated code
func splitSubdir(src string) (pkg, subdir string) {
	stop := len(src)
	if i := strings.Index(src, "?"); i > -1 {
		stop = i
	}
	var offset int
	if i := strings.Index(src[:stop], "://"); i > -1 {
		offset = i + 3
	}
	i := strings.Index(src[offset:stop], "//")
	if i == -1 {
		return src, ""
	}
	i += offset
	subdir = src[i+2:]
	pkg = src[:i]
	if j := strings.Index(subdir, "?"); j > -1 {
		pkg += subdir[j:]
		subdir = subdir[:j]
	}
	if subdir != "" {
		subdir = path.Clean(subdir)
	}
	return pkg, subdir
}

func computeGitPath(cacheDir, target string) (abs, bin string) {
	if _, err := exec.LookPath("git"); err != nil {
		exitErr("gorundir: git is not installed")
	}

	repoURL, ref, subdir := parseGitSource(target)

	repoAbs := filepath.Join(cacheDir, "git", normalize(repoURL+"/"+ref))
	err := os.MkdirAll(repoAbs, 0o755)
	check(err)

	abs = filepath.Join(repoAbs, subdir)
	bin = filepath.Join(cacheDir, "bin", normalize(repoURL+"/"+ref+"/"+subdir))

	err = exec.Command("git", "-C", repoAbs, "init", ".").Run()
	check(err)

	fetchCmd := []string{"git", "-C", repoAbs, "fetch", "--no-tags", "--depth=1", repoURL}
	if ref != "" {
		fetchCmd = append(fetchCmd, ref)
	}
	if output, err := exec.Command(fetchCmd[0], fetchCmd[1:]...).CombinedOutput(); err != nil {
		os.Stderr.Write(output)
		os.Stderr.WriteString("\n")
		exitErr("gorundir: cannot fetch " + repoURL + " on ref " + ref)
	}

	if output, err := exec.Command("git", "-C", repoAbs, "checkout", "-f", "FETCH_HEAD").CombinedOutput(); err != nil {
		os.Stderr.Write(output)
		os.Stderr.WriteString("\n")
		exitErr("gorundir: cannot checkout to " + ref)
	}

	return abs, bin
}
