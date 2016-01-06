package gta

import (
	"bufio"
	"fmt"
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
type Git struct{}

// Diff returns a set of changed directories.
func (g *Git) Diff() (map[string]bool, error) {
	// We get the current git sha
	out, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return nil, err
	}
	sha := strings.TrimSpace(string(out))

	// We get the root of the repository to build our full path.
	out, err = exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return nil, err
	}
	root := strings.TrimSpace(string(out))

	arg := fmt.Sprintf("origin/master...%s", sha)
	cmd := exec.Command("git", "diff", arg, "--name-only")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	dirs := map[string]bool{}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		filename := scanner.Text()

		// We build our full absolute file path.
		full, err := filepath.Abs(filepath.Join(root, filename))
		if err != nil {
			return nil, err
		}
		dirs[filepath.Dir(full)] = false
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return dirs, cmd.Wait()
}
