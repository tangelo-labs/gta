package embed

import _ "embed"

var (
	//go:embed files/prodfile
	_ string
)

type V struct{}
