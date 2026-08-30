package main

import (
	"fmt"
	"os"
	"strings"
)

type tag struct{ text string }

func (t tag) String() string { return "<" + t.text + ">" }

type wrapped struct {
	name string
	data []byte
}

func main() {
	bad := []byte{0x68, 0xff, 0x69}
	text := string(bad)

	fmt.Printf("%s|%s|%q\n", bad, text, text)
	fmt.Printf("%v|%v\n", text, bad)
	fmt.Printf("%x|%X\n", bad, bad)
	fmt.Printf("[%8s]\n", text)
	fmt.Println(text, len(text))
	fmt.Print(text, "\n")

	os.Stdout.Write(bad)
	os.Stdout.WriteString("\n")
	fmt.Fprintf(os.Stdout, "%s\n", bad)

	out := fmt.Sprintf("%s", bad)
	fmt.Println(len(out), []byte(out))
	fmt.Println(len(fmt.Sprint(text)), fmt.Sprint(text) == text)

	fmt.Printf("%s\n", tag{text: text})
	fmt.Printf("%v\n", wrapped{name: text, data: bad})
	fmt.Printf("%s\n", []string{text, "ok"})
	fmt.Printf("%s\n", map[string]string{"k": text})
	fmt.Printf("%s\n", fmt.Errorf("cause %s", text))

	fmt.Println(strings.ToUpper(text) == text, strings.Contains(text, "\xff"))

	// A rune loop reads the invalid byte as the replacement character, and the
	// byte itself keeps its value.
	for i, r := range text {
		fmt.Print(i, r, " ")
	}
	fmt.Println()
	for i := 0; i < len(text); i++ {
		fmt.Print(text[i], " ")
	}
	fmt.Println()
}
