package gta

import "testing"

func TestPackageContextImplementsPackager(t *testing.T) {
	var sut interface{} = new(packageContext)
	if _, ok := sut.(Packager); !ok {
		t.Error("expected to implement Packager")
	}
}
