package store

import "fmt"

// entry is unexported, and its 2 methods carry an exported name. An exported
// type embeds it, thus each name reaches another package.
type entry struct {
	Key   string
	count int
}

func (e *entry) Name() string { return e.Key }

func (e *entry) Bump() int {
	e.count++
	return e.count
}

func (e *entry) hidden() string { return "hidden " + e.Key }

// class is a Dart keyword and a valid Go name. The embedded field of this
// type thus needs the name of a public Dart field with the keyword suffix.
type class struct{ Tag int }

func (c class) Label() string { return fmt.Sprintf("tag %d", c.Tag) }

// Item embeds the 2 unexported types by value.
type Item struct {
	entry
	class
}

// Holder embeds an unexported type through a pointer.
type Holder struct {
	*entry
}

func NewItem(key string, tag int) *Item {
	return &Item{entry{Key: key}, class{Tag: tag}}
}

func NewHolder(key string) Holder { return Holder{&entry{Key: key}} }

// Inside reads the unexported method, which stays inside this package.
func Inside(i *Item) string { return i.hidden() }
