package main

import (
	"log"
	"os"
	"strings"
)

// The program writes each line to a buffer, thus no stamp of the clock reaches
// the output. A logger with no flag writes the prefix and the message only.

type buffer struct {
	sb strings.Builder
}

func (b *buffer) Write(p []byte) (int, error) {
	b.sb.Write(p)
	return len(p), nil
}

func main() {
	var b buffer
	l := log.New(&b, "app: ", 0)
	l.Print("plain")
	l.Printf("count %d", 3)
	l.Println("with", "words")
	l.SetPrefix("two: ")
	l.SetFlags(log.Lmsgprefix)
	l.Print("after the prefix")

	log.SetOutput(&b)
	log.SetFlags(0)
	log.SetPrefix("top: ")
	log.Print("through the package")
	log.Printf("%s=%d", "n", 7)

	if err := l.Output(1, "no newline"); err != nil {
		log.Println("output:", err)
	}
	os.Stdout.WriteString(b.sb.String())

	d := log.Default()
	d.SetOutput(&b)
	d.SetPrefix("")
	os.Stdout.WriteString(d.Prefix() + "|")
	os.Stdout.WriteString(strings.Repeat("-", 3) + "\n")
	os.Stdout.WriteString(logFlags())
}

func logFlags() string {
	l := log.New(os.Stdout, "", log.LstdFlags)
	if l.Flags() == log.Ldate|log.Ltime {
		return "flags ok\n"
	}
	return "flags bad\n"
}
