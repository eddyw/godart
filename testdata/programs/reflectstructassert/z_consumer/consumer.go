package consumer

import "fmt"

func Use(x any) { v, ok := x.(struct{ X int }); fmt.Println(ok, v.X) }

func UseOnly(x any) { v, ok := x.(struct{ Z int }); fmt.Println(ok, v.Z) }
