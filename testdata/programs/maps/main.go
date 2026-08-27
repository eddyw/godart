package main

type key struct {
	a int
	b string
}

func main() {
	m := make(map[string]int)
	m["one"] = 1
	m["two"] = 2
	println(len(m), m["one"], m["two"], m["none"])

	v, ok := m["one"]
	println(v, ok)
	v, ok = m["none"]
	println(v, ok)

	delete(m, "one")
	println(len(m), m["one"])

	lit := map[string]int{"a": 1, "b": 2}
	println(len(lit), lit["b"])

	total := 0
	for _, n := range lit {
		total += n
	}
	println(total)

	keys := 0
	for k := range lit {
		keys += len(k)
	}
	println(keys)

	structKeys := map[key]int{}
	structKeys[key{1, "x"}] = 10
	println(structKeys[key{1, "x"}], structKeys[key{2, "y"}])

	var nilMap map[string]int
	println(len(nilMap), nilMap == nil, nilMap["x"])

	counts := map[int]int{}
	for i := range 5 {
		counts[i%2]++
	}
	println(counts[0], counts[1])

	clear(lit)
	println(len(lit))
}
