package foreign

type Private interface{ hidden(int) }
type Public interface{ Visible(int) }

type Wrong int

func (Wrong) hidden() {}

type Right int

func (Right) hidden(int) {}
