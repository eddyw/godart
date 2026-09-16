package values

type Number int

func (n *Number) Add(v Number) { *n += v }

var Value Number = 3
var List = []int{1, 2}

func Pointer() *Number    { return &Value }
func ListPointer() *[]int { return &List }
func Set(v Number)        { Value = v }
func Increment()          { Value.Add(1) }
