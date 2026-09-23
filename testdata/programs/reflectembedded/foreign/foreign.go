package foreign

type Base int

func (Base) Get() int { return 7 }
func (Base) hidden()  {}

type Hidden interface{ hidden() }
