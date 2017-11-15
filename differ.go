package gta

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Differ implements a single method, used to note code diffs
// and the dirs they occur.
type Differ interface {
	// Diff returns a set of absolute pathed directories
	// that have files that have been modified.
	Diff() (map[string]bool, error)
}

// We check to make sure Git implements the Differ interface.
var _ Differ = &Git{}

// Git implements the Differ interface using a git version control method.
type Git struct {
	UseMergeCommit bool
}

// Diff returns a set of changed directories.
func (g *Git) Diff() (map[string]bool, error) {
	dirs, _, err := g.diffDirs()
	if err != nil {
		return nil, err
	}

	existsDirs := map[string]bool{}
	for dir := range dirs {
		if exists(dir) {
			existsDirs[dir] = false
		}
	}

	return existsDirs, nil
}

// DiffFiles returns a set of changed files.
func (g *Git) DiffFiles() (map[string]bool, error) {
	_, files, err := g.diffDirs()
	if err != nil {
		return nil, err
	}

	existsFiles := map[string]bool{}
	for file := range files {
		if exists(file) {
			existsFiles[file] = false
		}
	}

	return existsFiles, nil
}

// diffDirs returns a set of changed directories and files
func (g *Git) diffDirs() (map[string]bool, map[string]bool, error) {
	// We get the root of the repository to build our full path.
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return nil, nil, err
	}
	root := strings.TrimSpace(string(out))

	parent1 := "origin/master"
	parent2 := "HEAD"
	if g.UseMergeCommit {
		out, err := exec.Command("git", "log", "-1", "--merges", "--pretty=format:%p").Output()
		if err != nil {
			return nil, nil, err
		}
		parents := strings.TrimSpace(string(out))
		parentSplit := strings.Split(parents, " ")
		if len(parentSplit) != 2 {
			return nil, nil, fmt.Errorf("could not discover parent merge commits")
		}
		parent1 = parentSplit[0]
		parent2 = parentSplit[1]
	}

	cmd := exec.Command("git", "diff", fmt.Sprintf("%s...%s", parent1, parent2), "--name-only")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}

	dirs, files, err := diffFileDirectories(root, stdout)
	if err != nil {
		return nil, nil, err
	}

	return dirs, files, cmd.Wait()
}

// diffFileDirectories returns the directories and files that have changed
func diffFileDirectories(root string, r io.Reader) (map[string]bool, map[string]bool, error) {
	dirs := map[string]bool{}
	files := map[string]bool{}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		filename := scanner.Text()

		// We build our full absolute file path.
		full, err := filepath.Abs(filepath.Join(root, filename))
		if err != nil {
			return nil, nil, err
		}

		files[full] = false
		dirs[filepath.Dir(full)] = false
	}

	return dirs, files, scanner.Err()
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
