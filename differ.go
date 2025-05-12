/*
Copyright 2016 The gta AUTHORS. All rights reserved.

Use of this source code is governed by the Apache 2 license that can be found
in the LICENSE file.
*/
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

// A Differ implements provides methods that return values to understand the
// directories and files that have changed.
// and the dirs they in which occur.
type Differ interface {
	// Diff returns a set of absolute pathed directories that have files that
	// have been modified.
	Diff() (map[string]Directory, error)

	// DiffFiles returns a map whose keys are absolute files paths. A map value
	// is true when the file exists.
	DiffFiles() (map[string]bool, error)

	// DiffGoModDeps returns a map of dependencies that have been changed in go.mod.
	DiffGoModDeps() (map[string]struct{}, error)
}

// GitDifferOption is an option function used to modify a git differ
type GitDifferOption func(*git)

// SetUseMergeCommit sets the useMergeCommit field on a git differ
func SetUseMergeCommit(useMergeCommit bool) GitDifferOption {
	return func(gd *git) {
		gd.useMergeCommit = useMergeCommit
	}
}

// SetBaseBranch sets the baseBranch field on a git differ
func SetBaseBranch(baseBranch string) GitDifferOption {
	return func(gd *git) {
		gd.baseBranch = baseBranch
	}
}

// SetUseHeadToHead sets the useHeadToHead field on a git differ
func SetUseHeadToHead(useHeadToHead bool) GitDifferOption {
	return func(gd *git) {
		gd.useHeadToHead = useHeadToHead
	}
}

// NewGitDiffer returns a Differ that determines differences using git.
func NewGitDiffer(opts ...GitDifferOption) Differ {
	g := &git{
		baseBranch: "origin/master",
	}

	for _, opt := range opts {
		opt(g)
	}

	return &differ{
		diff:     g.diff,
		depsDiff: g.fetchGoModDepChanges,
	}
}

// NewFileDiffer returns a Differ that operates on a list of absolute paths of
// changed files.
func NewFileDiffer(files []string) Differ {
	m := make(map[string]struct{}, len(files))

	for _, v := range files {
		m[v] = struct{}{}
	}

	return &differ{
		diff:     func() (map[string]struct{}, error) { return m, nil },
		depsDiff: func() (map[string]struct{}, error) { return make(map[string]struct{}), nil },
	}
}

type differ struct {
	diff     func() (map[string]struct{}, error)
	depsDiff func() (map[string]struct{}, error)
}

// git implements the Differ interface using a git version control method.
type git struct {
	baseBranch     string
	useMergeCommit bool
	useHeadToHead  bool
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
func (d *differ) Diff() (map[string]Directory, error) {
	files, err := d.diff()
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
func (d *differ) DiffFiles() (map[string]bool, error) {
	files, err := d.diff()
	if err != nil {
		return nil, err
	}

	existsFiles := map[string]bool{}
	for abs := range files {
		existsFiles[abs] = exists(abs)
	}

	return existsFiles, nil
}

// DiffGoModDeps returns a set of changed dependencies in go.mod.
func (d *differ) DiffGoModDeps() (map[string]struct{}, error) {
	return d.depsDiff()
}

func (g *git) getMergeParents() (parent1 string, rightwardParents []string, err error) {
	out, err := execWithStderr(exec.Command("git", "log", "-1", "--pretty=format:%p"))
	if err != nil {
		return
	}
	parents := strings.TrimSpace(string(out))
	parentSplit := strings.Split(parents, " ")

	// for merge commits, parents will include both values
	if len(parentSplit) >= 2 {
		parent1 = parentSplit[0]
		rightwardParents = parentSplit[1:]
		return
	}

	// for squash-merge/rebase commits, get the most recent merge commit hash and use as left parent
	out, err = execWithStderr(exec.Command("git", "log", "-1", "--merges", "--pretty=format:%h"))
	if err != nil {
		return
	}
	parent1 = strings.TrimSpace(string(out))
	rightwardParents = []string{"HEAD"}
	return
}

// diff returns a set of changed files.
func (g *git) diff() (map[string]struct{}, error) {
	g.onceDiff.Do(func() {
		files, err := func() (map[string]struct{}, error) {
			// We get the root of the repository to build our full path.
			root, err := g.root()
			if err != nil {
				return nil, err
			}

			parent1, rightwardParents, err := g.getParents()
			if err != nil {
				return nil, fmt.Errorf("git differ failed to get branch parents when getting go.mod dependency changes: %w", err)
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
			g.diffErr = err
			return
		}

		g.changedFiles = files
	})

	return g.changedFiles, g.diffErr
}

func (g *git) fetchGoModDepChanges() (map[string]struct{}, error) {
	filesChanged, err := g.diff()
	if err != nil {
		return nil, fmt.Errorf("git differ failed to get files changed when getting go.mod dependency changes: %w", err)
	}

	// We get the root of the repository to build our full path.
	root, err := g.root()
	if err != nil {
		return nil, fmt.Errorf("git differ failed to get root path when getting go.mod dependency changes: %w", err)
	}

	// Get the absolute path of go.mod
	goModPath, err := filepath.Abs(filepath.Join(root, "go.mod"))
	if err != nil {
		return nil, fmt.Errorf("git differ failed to get absolute path of go.mod when getting go.mod dependency changes: %w", err)
	}

	// Look up if go.mod is in the changed files
	_, found := filesChanged[goModPath]
	if !found {
		return map[string]struct{}{}, nil
	}

	parent1, rightwardParents, err := g.getParents()
	if err != nil {
		return nil, fmt.Errorf("git differ failed to get branch parents when getting go.mod dependency changes: %w", err)
	}

	changes := make(map[string]struct{})

	// Loop the commit parents to get all deps changed in go.mod
	for _, parent2 := range rightwardParents {
		// get go.mod changes
		out, err := execWithStderr(exec.Command("git", "--no-pager", "diff", "--unified=0", fmt.Sprintf("%s...%s", parent1, parent2), "go.mod"))
		if err != nil {
			return nil, fmt.Errorf("git differ failed to go.mod when getting go.mod dependency changes: %w", err)
		}

		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			changeDetected := strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++")

			if changeDetected {
				if dep, ok := isDependency(line); ok {
					changes[dep] = struct{}{}
				}

				if rep, ok := isReplace(line); ok {
					changes[rep] = struct{}{}
				}
			}
		}
	}

	return changes, nil
}

func (g *git) getParents() (parent1 string, rightwardParents []string, errR error) {
	parent1 = g.baseBranch
	rightwardParents = []string{"HEAD"}

	// When HeadToHead is not set, vanilla behavior. Get root commit when the branch was created from the base as the parent.
	if !g.useHeadToHead {
		// get the revision from which HEAD was branched from g.baseBranch.
		resParent1, err := g.branchPointOf("HEAD")
		if err != nil {
			errR = err

			return
		}

		// If the branch point is unknown, fall back to using the base branch. In
		// most cases, this will be fine, but results in a corner case when base
		// branch has been merged into the branch since branch was created. In
		// that case, the differences from the base branch and the most recent
		// merge will not be considered.
		if resParent1 != "" {
			parent1 = resParent1
		}
	}

	if g.useMergeCommit {
		resParent1, resRightwardParents, err := g.getMergeParents()
		if err != nil {
			errR = err
			return
		}

		parent1, rightwardParents = resParent1, resRightwardParents
	}

	return
}

func (g *git) root() (string, error) {
	out, err := execWithStderr(exec.Command("git", "rev-parse", "--show-toplevel"))
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
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

// branchPointOf will return the oldest commit on g.baseBranch that is in
// branch. If no such commit exists (e.g. branch is a shallow clone or branch
// does not share history with g.baseBranch), then an empty string is returned.
func (g *git) branchPointOf(branch string) (string, error) {
	// Use --topo-order to ensure graph order is respected.
	//
	// Use --parents so each line will list the commit and its parents.
	//
	// Use --reverse so the first commit in the output will be the oldest commit.
	// branch that is not on the base branch.
	//
	// Do NOT use --first-parent, because the branch may have had merges from
	// other branches into it, and we want the oldest possible branch point
	// from the base branch in branch.
	//
	// Do NOT try using git merge-base at all. It would not deliver the right
	// result when g.baseBranch had been merged into branch sometime after branch
	// was created from g.baseBranch. In such a case, the merge base would be the
	// the merge commit where g.baseBranch was merged into branch.
	out, err := execWithStderr(exec.Command("git", "rev-list", "--topo-order", "--parents", "--reverse", branch, "^"+g.baseBranch))
	if err != nil {
		return "", nil
	}

	lines := strings.Split(string(out), "\n")
	firstCommit := lines[0]
	ancestors := strings.Fields(firstCommit)
	if len(ancestors) < 2 {
		return "", nil
	}
	branchPoint := ancestors[1]
	return branchPoint, nil
}

func isDependency(line string) (string, bool) {
	line = strings.TrimSpace(line)

	parts := strings.Fields(line)
	if len(parts) == 3 {
		return parts[0], true
	}

	if strings.HasPrefix(line, "replace ") {
		line = strings.TrimSpace(strings.ReplaceAll(line, "replace ", ""))
		parts := strings.Fields(line)

		return parts[0], true
	}

	return "", false
}

func isReplace(line string) (string, bool) {
	line = strings.TrimSpace(line)

	parts := strings.Split(line, "=>")
	if len(parts) == 2 {
		return parts[0], true
	}

	if strings.HasPrefix(line, "replace ") {
		line = strings.TrimSpace(strings.ReplaceAll(line, "replace ", ""))
		parts := strings.Split(line, "=>")

		return parts[0], true
	}

	return "", false
}

type fileDiffer struct {
	changedFiles map[string]struct{}
}

func execWithStderr(c *exec.Cmd) (out []byte, err error) {
	var stderr strings.Builder
	c.Stderr = &stderr
	out, err = c.Output()
	if err != nil {
		err = fmt.Errorf("%w: %s", err, stderr.String())
	}
	return
}
