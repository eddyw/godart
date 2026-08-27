// Package greet builds a greeting.
package greet

type Style int

const (
	Plain Style = iota
	Loud
)

// Greeter holds the state of a greeting.
type Greeter struct {
	Name  string
	style Style
}

func New(name string) *Greeter {
	return &Greeter{Name: name}
}

func (g *Greeter) SetStyle(s Style) {
	g.style = s
}

func (g *Greeter) Text() string {
	if g.style == Loud {
		return "HELLO, " + g.Name
	}
	return "hello, " + g.Name
}

func Twice(name string) string {
	g := New(name)
	return g.Text() + " " + g.Text()
}
