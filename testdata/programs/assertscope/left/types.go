package twin

type Number int
type Method int

func (Method) M() {}

func Private() any { return struct{ hidden int }{} }
func Promoted() any {
	return struct {
		Method
		hidden int
	}{}
}
