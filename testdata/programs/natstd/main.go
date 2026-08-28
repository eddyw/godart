package main

import (
	"cmp"
	"container/list"
	"fmt"
	"unicode"
)

func main() {
	fmt.Println(unicode.IsLetter('a'), unicode.IsDigit('7'), unicode.IsSpace(' '))
	fmt.Println(string(unicode.ToUpper('é')), string(unicode.ToLower('Q')))
	fmt.Println(unicode.IsUpper('A'), unicode.IsLower('a'), unicode.IsPunct(','))
	fmt.Println(unicode.In('5', unicode.Nd), unicode.Is(unicode.Han, '世'))
	fmt.Println(unicode.SimpleFold('a'), string(unicode.ToTitle('a')))

	fmt.Println(cmp.Compare(1, 2), cmp.Compare(2, 2), cmp.Compare(3, 2))
	fmt.Println(cmp.Less("a", "b"), cmp.Or(0, 0, 3), cmp.Or("", "x"))

	l := list.New()
	l.PushBack(1)
	l.PushBack(2)
	l.PushFront(0)
	for e := l.Front(); e != nil; e = e.Next() {
		fmt.Print(e.Value, " ")
	}
	fmt.Println(l.Len())

	other := list.New()
	other.PushBack("a")
	other.PushBack("b")
	l.PushBackList(other)
	fmt.Println(l.Len(), l.Back().Value)

	l.Remove(l.Front())
	fmt.Println(l.Len(), l.Front().Value)

	mid := l.Front().Next()
	l.MoveToFront(mid)
	fmt.Println(l.Front().Value)
	l.InsertAfter(9, l.Front())
	fmt.Println(l.Front().Next().Value, l.Len())
	for e := l.Back(); e != nil; e = e.Prev() {
		fmt.Print(e.Value, " ")
	}
	fmt.Println()
}
