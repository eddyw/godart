package api

import _ "unsafe"

//go:linkname twice example.com/linkfunc/impl.twice
func twice(n int) int

//go:linkname wait example.com/linkfunc/impl.wait
func wait(n int) int

var Saved = twice

func Call(n int) int { return twice(n) }
func Wait(n int) int { return wait(n) }

//go:linkname local example.com/linkfunc/api.direct
func local(n int) int
func direct(n int) int { return n + 3 }
func Local(n int) int  { return local(n) }
