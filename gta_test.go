package gta

import (
	"errors"
	"go/build"
	"reflect"
	"testing"
)

var _ Differ = &testDiffer{}

type testDiffer struct {
	diff map[string]bool
}

func (t *testDiffer) Diff() (map[string]bool, error) {
	return t.diff, nil
}

var _ Packager = &testPackager{}

type testPackager struct {
	dirs2Imports map[string]string
	graph        *Graph
}

func (t *testPackager) PackageFromDir(a string) (*build.Package, error) {
	path, ok := t.dirs2Imports[a]
	if !ok {
		return nil, errors.New("dir not found")
	}
	return &build.Package{
		ImportPath: path,
	}, nil
}

func (t *testPackager) PackageFromImport(a string) (*build.Package, error) {
	for _, v := range t.dirs2Imports {
		if a == v {
			return &build.Package{
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
		diff: map[string]bool{
			"dirC": false,
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
	}
	want := []*build.Package{
		&build.Package{ImportPath: "A"},
		&build.Package{ImportPath: "B"},
		&build.Package{ImportPath: "C"},
	}

	gta, err := New(SetDiffer(difr), SetPackager(pkgr))
	if err != nil {
		t.Fatal(err)
	}

	got, err := gta.DirtyPackages()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(want, got) {
		t.Errorf("want: %v", want)
		t.Errorf(" got: %v", got)
		t.Fatal("expected want and got to be equal")
	}
}
