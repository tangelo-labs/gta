package gta

import (
	"bufio"
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
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return nil, err
	}
	root := strings.TrimSpace(string(out))

	cmd := exec.Command("git", "diff", "origin/master", "--name-only")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Start()

	dirs := map[string]bool{}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		filename := scanner.Text()

		// we build our full absolute file path
		full, err := filepath.Abs(filepath.Join(root, filename))
		if err != nil {
			return nil, err
		}
		dirs[filepath.Dir(full)] = false
	}

	return dirs, scanner.Err()
}
