package foreign

type Value struct {
	X int `foreign`
}

func (v *Value) Add(n int) { v.X += n }
func (v Value) Read() int  { return v.X + 2 }
