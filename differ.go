package gta

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// Differ implements a single method, used to note code diffs
// and the dirs they occur.
type Differ interface {
	// Diff returns a set of absolute pathed directories that have files that
	// have been modified.
	Diff() (map[string]Directory, error)

	// DiffFiles returns a set of absolute pathed files that have been modified.
	DiffFiles() (map[string]bool, error)
}

// NewDiffer returns a Differ that determines differences using git.
func NewDiffer(useMergeCommit bool) Differ {
	return &git{
		useMergeCommit: useMergeCommit,
	}
}

// git implements the Differ interface using a git version control method.
type git struct {
	useMergeCommit bool
	onceDiff       sync.Once
	changedFiles   map[string]struct{}
	diffErr        error
}

// A Directory describes changes to a directory and its contents.
type Directory struct {
	Exists bool
	Files  []string
}

// Diff returns a set of changed directories. The keys of the returned map are
// absolute paths.
func (g *git) Diff() (map[string]Directory, error) {
	files, err := g.diff()
	if err != nil {
		return nil, err
	}

	existsDirs := make(map[string]Directory, len(files))
	for abs := range files {
		absdir := filepath.Dir(abs)
		dir, ok := existsDirs[absdir]
		if !ok {
			dir.Exists = exists(absdir)
		}

		fn := filepath.Base(abs)
		dir.Files = append(dir.Files, fn)
		existsDirs[absdir] = dir
	}

	return existsDirs, nil
}

// DiffFiles returns a set of changed files. The keys of the returned map are
// absolute paths. The map values indicate whether or not the file exists: a
// false value means the file was deleted.
func (g *git) DiffFiles() (map[string]bool, error) {
	files, err := g.diff()
	if err != nil {
		return nil, err
	}

	existsFiles := map[string]bool{}
	for abs := range files {
		existsFiles[abs] = exists(abs)
	}

	return existsFiles, nil
}

// diff returns a set of changed files.
func (g *git) diff() (map[string]struct{}, error) {
	g.onceDiff.Do(func() {
		files, err := func() (map[string]struct{}, error) {
			// We get the root of the repository to build our full path.
			out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
			if err != nil {
				return nil, err
			}
			root := strings.TrimSpace(string(out))
			parent1 := "origin/master"
			rightwardParents := []string{"HEAD"}
			if g.useMergeCommit {
				out, err := exec.Command("git", "log", "-1", "--merges", "--pretty=format:%p").Output()
				if err != nil {
					return nil, err
				}
				parents := strings.TrimSpace(string(out))
				parentSplit := strings.Split(parents, " ")
				if len(parentSplit) < 2 {
					return nil, fmt.Errorf("could not discover parent merge commits")
				}
				parent1 = parentSplit[0]
				rightwardParents = parentSplit[1:]
			}

			files := make(map[string]struct{})

			for _, parent2 := range rightwardParents {
				// get the names of all affected files without doing rename detection.
				cmd := exec.Command("git", "diff", fmt.Sprintf("%s...%s", parent1, parent2), "--name-only", "--no-renames")
				stdout, err := cmd.StdoutPipe()
				if err != nil {
					return nil, err
				}

				if err := cmd.Start(); err != nil {
					return nil, err
				}

				changedPaths, err := diffPaths(root, stdout)
				if err != nil {
					return nil, err
				}

				for path := range changedPaths {
					files[path] = struct{}{}
				}

				err = cmd.Wait()
				if err != nil {
					return nil, err
				}
			}
			return files, nil
		}()
		if err != nil {
			g.diffErr = nil
			return
		}

		g.changedFiles = files
	})

	return g.changedFiles, g.diffErr
}

// diffPaths returns the path that have changed.
func diffPaths(root string, r io.Reader) (map[string]struct{}, error) {
	paths := make(map[string]struct{})

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		path := scanner.Text()

		// We build our full absolute file path.
		full, err := filepath.Abs(filepath.Join(root, path))
		if err != nil {
			return nil, err
		}

		paths[full] = struct{}{}
	}

	return paths, scanner.Err()
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
