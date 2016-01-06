package gta

import (
	"os/exec"
	"path/filepath"
	"strings"
)

type Differ interface {
	// Diff returns a set of absolute pathed directories
	// that have files that have been modified
	Diff() (map[string]bool, error)
}

// check to make sure Git implements the Differ interface
var _ Differ = &Git{}

// Git implements the Differ interface using a git version control method
type Git struct{}

// Diff returns a set of changed directories
func (g *Git) Diff() (map[string]bool, error) {
	// We get the root of the repository to build our full path
	cmd := []string{"rev-parse", "--show-toplevel"}
	out, err := exec.Command("git", cmd...).Output()
	if err != nil {
		return nil, err
	}
	root := strings.TrimSpace(string(out))

	// git diff all files from _master_
	cmd = []string{"diff", "origin/master", "--name-only"}
	out, err = exec.Command("git", cmd...).Output()
	if err != nil {
		return nil, err
	}

	changed := strings.Split(string(out), "\n")
	dirs := map[string]bool{}
	for _, filename := range changed {
		if filename == "" {
			continue
		}

		// we build our full absolute file path
		full, err := filepath.Abs(filepath.Join(root, filename))
		if err != nil {
			return nil, err
		}
		dirs[filepath.Dir(full)] = false
	}

	return dirs, nil
}
