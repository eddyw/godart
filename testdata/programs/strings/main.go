package main

func main() {
	s := "hello"
	println(len(s), s[0])
	println(s + ", world")
	println(s < "world", s == "hello", s > "abc")
	println(s[1:3])

	u := "héllo"
	println(len(u))
	for i, r := range u {
		println(i, r)
	}

	b := []byte(s)
	b[0] = 72
	println(string(b))

	r := []rune(u)
	println(len(r), string(r))
	println(string(rune(233)))

	var empty string
	println(len(empty), empty == "")
}
