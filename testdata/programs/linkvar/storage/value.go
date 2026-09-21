package storage

var count = 7
var state = struct{ N int }{3}

func init()          { count++ }
func Read() int      { return count }
func Address() *int  { return &count }
func State() int     { return state.N }
func SetState(n int) { state.N = n }
