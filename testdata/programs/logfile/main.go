package main

import (
	"log"
	"os"
)

func main() {
	l := log.New(os.Stdout, "app: ", log.Lshortfile)
	l.Print("a line")
	l.Printf("a %s line", "second")
	l.Output(1, "a third line")

	m := log.New(os.Stdout, "tag: ", log.Lshortfile|log.Lmsgprefix)
	m.Println("a fourth line")
}
