package types

type Source struct {
	N      int
	hidden [2]int
	Values []int
}
type Target Source

func New() Source            { return Source{3, [2]int{4, 5}, []int{6}} }
func (v Source) Number() int { return v.N + v.hidden[0] }
func (v Target) Number() int { return v.N + v.hidden[1] }
