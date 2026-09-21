package other

import (
	_ "example.com/linkvar/storage"
	_ "unsafe"
)

//go:linkname number example.com/linkvar/storage.count
var number int

func Address() *int { return &number }
func Read() int     { return number }
