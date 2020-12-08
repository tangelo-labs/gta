/*
Copyright 2016 The gta AUTHORS. All rights reserved.

Use of this source code is governed by the Apache 2 license that can be found
in the LICENSE file.
*/
package gta

import (
	"encoding/json"
	"errors"
	"go/build"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var _ Differ = &testDiffer{}

type testDiffer struct {
	diff map[string]Directory
}

func (t *testDiffer) Diff() (map[string]Directory, error) {
	return t.diff, nil
}

func (t *testDiffer) DiffFiles() (map[string]bool, error) {
	panic("not implemented")
}

var _ Packager = &testPackager{}

type testPackager struct {
	dirs2Imports map[string]string
	graph        *Graph
	errs         map[string]error
}

func (t *testPackager) PackageFromDir(a string) (*Package, error) {
	// we pass back an err
	err, eok := t.errs[a]
	if eok {
		return nil, err
	}

	path, ok := t.dirs2Imports[a]
	if !ok {
		return nil, errors.New("dir not found")
	}

	return &Package{
		ImportPath: path,
	}, nil
}

func (t *testPackager) PackageFromEmptyDir(a string) (*Package, error) {
	return nil, errors.New("not implemented")
}

func (t *testPackager) PackageFromImport(a string) (*Package, error) {
	for _, v := range t.dirs2Imports {
		if a == v {
			return &Package{
				ImportPath: a,
			}, nil
		}
	}
	return nil, errors.New("pkg not found")
}

func (t *testPackager) DependentGraph() (*Graph, error) {
	return t.graph, nil
}

func TestGTA(t *testing.T) {
	// A depends on B depends on C
	// dirC is dirty, we expect them all to be marked
	difr := &testDiffer{
		diff: map[string]Directory{
			"dirC": Directory{Exists: true},
		},
	}

	graph := &Graph{
		graph: map[string]map[string]bool{
			"C": map[string]bool{
				"B": true,
			},
			"B": map[string]bool{
				"A": true,
			},
		},
	}

	pkgr := &testPackager{
		dirs2Imports: map[string]string{
			"dirA": "A",
			"dirB": "B",
			"dirC": "C",
		},
		graph: graph,
		errs:  make(map[string]error),
	}
	want := []Package{
		Package{ImportPath: "A"},
		Package{ImportPath: "B"},
		Package{ImportPath: "C"},
	}

	gta, err := New(SetDiffer(difr), SetPackager(pkgr))
	if err != nil {
		t.Fatal(err)
	}

	pkgs, err := gta.ChangedPackages()
	if err != nil {
		t.Fatal(err)
	}

	got := pkgs.AllChanges

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("(-want, +got)\n%s", diff)
	}
}

func TestGTA_ChangedPackages(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		// A depends on B depends on C
		// D depends on B
		// E depends on F depends on G

		difr := &testDiffer{
			diff: map[string]Directory{
				"dirC": Directory{Exists: true},
				"dirH": Directory{Exists: true},
			},
		}

		graph := &Graph{
			graph: map[string]map[string]bool{
				"C": map[string]bool{
					"B": true,
				},
				"B": map[string]bool{
					"A": true,
					"D": true,
				},
				"G": map[string]bool{
					"F": true,
				},
				"F": map[string]bool{
					"E": true,
				},
			},
		}

		pkgr := &testPackager{
			dirs2Imports: map[string]string{
				"dirA": "A",
				"dirB": "B",
				"dirC": "C",
				"dirD": "D",
				"dirF": "E",
				"dirG": "F",
				"dirH": "G",
			},
			graph: graph,
			errs:  make(map[string]error),
		}

		want := &Packages{
			Dependencies: map[string][]Package{
				"C": []Package{
					{ImportPath: "A"},
					{ImportPath: "B"},
					{ImportPath: "D"},
				},
				"G": []Package{
					{ImportPath: "E"},
					{ImportPath: "F"},
				},
			},
			Changes: []Package{
				{ImportPath: "C"},
				{ImportPath: "G"},
			},
			AllChanges: []Package{
				{ImportPath: "A"},
				{ImportPath: "B"},
				{ImportPath: "C"},
				{ImportPath: "D"},
				{ImportPath: "E"},
				{ImportPath: "F"},
				{ImportPath: "G"},
			},
		}

		gta, err := New(SetDiffer(difr), SetPackager(pkgr))
		if err != nil {
			t.Fatal(err)
		}

		got, err := gta.ChangedPackages()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("(-want, +got)\n%s", diff)
		}
	})
}

func TestGTA_Prefix(t *testing.T) {
	// A depends on B and foo
	// B depends on C and bar
	// C depends on qux
	difr := &testDiffer{
		diff: map[string]Directory{
			"dirB":   Directory{Exists: true},
			"dirC":   Directory{Exists: true},
			"dirFoo": Directory{Exists: true},
		},
	}

	graph := &Graph{
		graph: map[string]map[string]bool{
			"C": map[string]bool{
				"B": true,
			},
			"B": map[string]bool{
				"A": true,
			},
			"foo": map[string]bool{
				"A": true,
			},
			"bar": map[string]bool{
				"B": true,
			},
			"qux": map[string]bool{
				"C": true,
			},
		},
	}

	pkgr := &testPackager{
		dirs2Imports: map[string]string{
			"dirA":   "A",
			"dirB":   "B",
			"dirC":   "C",
			"dirFoo": "foo",
			"dirBar": "bar",
			"dirQux": "qux",
		},
		graph: graph,
		errs:  make(map[string]error),
	}
	want := []Package{
		Package{ImportPath: "C"},
		Package{ImportPath: "foo"},
	}

	gta, err := New(SetDiffer(difr), SetPackager(pkgr), SetPrefixes("foo", "C"))
	if err != nil {
		t.Fatal(err)
	}

	pkgs, err := gta.ChangedPackages()
	if err != nil {
		t.Fatal(err)
	}

	got := pkgs.AllChanges

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("(-want, +got)\n%s", diff)
	}
}

func TestNoBuildableGoFiles(t *testing.T) {
	// we have changes but they don't belong to any dirty golang files, so no dirty packages
	const dir = "docs"
	difr := &testDiffer{
		diff: map[string]Directory{
			dir: Directory{},
		},
	}

	pkgr := &testPackager{
		errs: map[string]error{
			dir: &build.NoGoError{
				Dir: dir,
			},
		},
	}

	var want []Package

	gta, err := New(SetDiffer(difr), SetPackager(pkgr))
	if err != nil {
		t.Fatal(err)
	}

	pkgs, err := gta.ChangedPackages()
	if err != nil {
		t.Fatal(err)
	}

	got := pkgs.AllChanges

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("(-want, +got)\n%s", diff)
	}
}

func TestSpecialCaseDirectory(t *testing.T) {
	// We want to ignore the special case directory "testdata"
	const (
		special1 = "specia/case/testdata"
		special2 = "specia/case/testdata/multi"
	)
	difr := &testDiffer{
		diff: map[string]Directory{
			special1: Directory{Exists: true}, // this
			special2: Directory{Exists: true},
			"dirC":   Directory{Exists: true},
		},
	}
	graph := &Graph{
		graph: map[string]map[string]bool{
			"C": map[string]bool{
				"B": true,
			},
			"B": map[string]bool{
				"A": true,
			},
		},
	}

	pkgr := &testPackager{
		dirs2Imports: map[string]string{
			"dirA": "A",
			"dirB": "B",
			"dirC": "C",
		},
		graph: graph,
		errs: map[string]error{
			special1: &build.NoGoError{
				Dir: special1,
			},
			special2: &build.NoGoError{
				Dir: special2,
			},
		},
	}

	want := []Package{
		Package{ImportPath: "A"},
		Package{ImportPath: "B"},
		Package{ImportPath: "C"},
	}

	gta, err := New(SetDiffer(difr), SetPackager(pkgr))
	if err != nil {
		t.Fatal(err)
	}

	pkgs, err := gta.ChangedPackages()
	if err != nil {
		t.Fatal(err)
	}

	got := pkgs.AllChanges

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("(-want, +got)\n%s", diff)
	}
}

func TestUnmarshalJSON(t *testing.T) {
	want := &Packages{
		Dependencies: map[string][]Package{
			"do/tools/build/gta": []Package{
				{
					ImportPath: "do/tools/build/gta/cmd/gta",
				},
				{
					ImportPath: "do/tools/build/gtartifacts",
				},
			},
		},
		Changes: []Package{
			{
				ImportPath: "do/teams/compute/octopus",
			},
		},
		AllChanges: []Package{
			{
				ImportPath: "do/teams/compute/octopus",
			},
		},
	}

	in := []byte(`{"dependencies":{"do/tools/build/gta":["do/tools/build/gta/cmd/gta","do/tools/build/gtartifacts"]},"changes":["do/teams/compute/octopus"],"all_changes":["do/teams/compute/octopus"]}`)

	got := new(Packages)
	err := json.Unmarshal(in, got)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("(-want, +got)\n%s", diff)
	}
}

func TestJSONRoundtrip(t *testing.T) {
	want := &Packages{
		Dependencies: map[string][]Package{
			"do/tools/build/gta": []Package{
				{
					ImportPath: "do/tools/build/gta/cmd/gta",
				},
				{
					ImportPath: "do/tools/build/gtartifacts",
				},
			},
		},
		Changes: []Package{
			{
				ImportPath: "do/teams/compute/octopus",
			},
		},
		AllChanges: []Package{
			{
				ImportPath: "do/teams/compute/octopus",
			},
		},
	}

	b, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}

	got := new(Packages)
	err = json.Unmarshal(b, got)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("(-want, +got)\n%s", diff)
	}
}
