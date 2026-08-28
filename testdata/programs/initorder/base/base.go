// Package base holds package variables whose order comes from the dependency,
// and not from the declaration.
package base

var Total = half * 2

var half = seed + 1

var seed = 4

// Label reads Suffix through a call. Go gives Suffix its value first.
var Label = build()

func build() string { return "base" + Suffix }

var Suffix = "-v1"
