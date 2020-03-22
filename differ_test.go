/*
Copyright 2016 The gta AUTHORS. All rights reserved.

Use of this source code is governed by the Apache 2 license that can be found
in the LICENSE file.
*/
package gta

import (
	"bytes"
	"reflect"
	"testing"
)

// check to make sure Git implements the Differ interface.
var _ Differ = &differ{}

func Test_diffFileDirectories(t *testing.T) {
	var tests = []struct {
		desc string
		root string
		buf  []byte
		want map[string]struct{}
	}{
		{
			desc: "single changed file",
			root: "/",
			buf:  []byte("foo/bar.go\n"),
			want: map[string]struct{}{
				"/foo/bar.go": struct{}{},
			},
		},
		{
			desc: "multiple changed files in same directory (duplicate)",
			root: "/foo",
			buf: []byte(`bar/bar.go
bar/baz.go`),
			want: map[string]struct{}{
				"/foo/bar/bar.go": struct{}{},
				"/foo/bar/baz.go": struct{}{},
			},
		},
		{
			desc: "multiple changed files in different directories",
			root: "/foo/bar",
			buf: []byte(`baz/bar.go
baz/qux/baz.go`),
			want: map[string]struct{}{
				"/foo/bar/baz/bar.go":     struct{}{},
				"/foo/bar/baz/qux/baz.go": struct{}{},
			},
		},
		{
			desc: "multiple changed files in different directories, with duplicate directories",
			root: "/",
			buf: []byte(`foo/bar.go
foo/baz.go
bar/foo.go
bar/baz/qux.go
bar/baz/corge.go
bar/baz/qux/corge.go
`),
			want: map[string]struct{}{
				"/foo/bar.go":           struct{}{},
				"/foo/baz.go":           struct{}{},
				"/bar/foo.go":           struct{}{},
				"/bar/baz/qux.go":       struct{}{},
				"/bar/baz/corge.go":     struct{}{},
				"/bar/baz/qux/corge.go": struct{}{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {

			got, err := diffPaths(tt.root, bytes.NewReader(tt.buf))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.want, got; !reflect.DeepEqual(want, got) {
				t.Fatalf("unexpected file directory map:\n- want: %v\n-  got: %v",
					want, got)
			}
		})
	}
}
