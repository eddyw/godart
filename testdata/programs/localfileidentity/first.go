package main

func first() (any, any) {
//line shared.go:1
	type Number int
	type Record struct{ Value int }
	return Number(1), Record{1}
}
