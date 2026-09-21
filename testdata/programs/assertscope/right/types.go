package twin

type Number int
type Method int

func (Method) M() {}

func AssertPrivate(x any) { _ = x.(struct{ hidden int }) }
func AssertPromoted(x any) {
	_ = x.(struct {
		Method
		hidden int
	})
}
