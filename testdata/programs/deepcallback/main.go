package main

import "fmt"

//gart:sync
type Callback func() bool

func walk(n int) int {
	if n == 0 {
		return 0
	}
	return 1 + walk(n-1)
}
func fail(n int) {
	if n == 0 {
		panic("deep")
	}
	fail(n - 1)
}
func HostCallback() Callback { return func() bool { return walk(20000) == 20000 } }

type result struct{ text string }

//gart:sync
type Formatter func(int) result
type depth int

func (n depth) String() string {
	if n == 0 {
		return "ok"
	}
	return (n - 1).String()
}
func main() {
	var format Formatter = func(n int) result { return result{fmt.Sprintf("number %d", walk(n))} }
	fmt.Println(format(20000).text)
	ok := true
	for i := 0; i < 2000; i++ {
		ok = ok && format(1).text == "number 1"
	}
	fmt.Println(ok, fmt.Sprint(depth(20000)))

	fmt.Println(HostCallback()())
	fmt.Println(walk(20000))
	count := 0
	var callback Callback = func() bool { count = walk(20000); return count == 20000 }
	fmt.Println(invoke(callback), count)
	fmt.Println(invoke(Callback(func() bool { return invoke(Callback(func() bool { return walk(15000) == 15000 })) })))
	fmt.Println(invoke(Callback(func() (ok bool) { defer func() { ok = recover() == "deep" }(); fail(12000); return false })))
	var empty Callback
	fmt.Println(empty == nil)
}

func invoke(fn Callback) bool { return fn() }
