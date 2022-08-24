package embed_test

import (
	"testing"

	_ "embed"
)

var (
	//go:embed files/testfile
	_ string
)

func TestV(t *testing.T) {
	t.Log(foo.V())
}
