package main

import (
	"example.com/multi/count"
	"example.com/multi/greet"
)

func main() {
	g := greet.New("world")
	println(g.Text())
	g.SetStyle(greet.Loud)
	println(g.Text())
	println(greet.Twice("go"))
	println(count.Runes("héllo"))
	println(int(greet.Plain), int(greet.Loud))
}
