package bridge

import (
	_ "example.com/linkvar/other"
	_ "unsafe"
)

//go:linkname number example.com/linkvar/storage.count
var number int

func Address() *int { return &number }
