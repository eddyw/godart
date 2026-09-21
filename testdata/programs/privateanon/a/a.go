package a

func Make(n int) struct{ n int }    { return struct{ n int }{n} }
func Read(v struct{ n int }) int    { return v.n }
func Set(v *struct{ n int }, n int) { v.n = n }
func Accept(v any) bool {
	_, ok := v.(struct{ n int })
	return ok
}

type Box struct{ Value struct{ n int } }

type hidden struct{ n int }

func Nested(n int) struct {
	Inner struct{ n int }
	value hidden
} {
	return struct {
		Inner struct{ n int }
		value hidden
	}{Make(n), hidden{n + 1}}
}
