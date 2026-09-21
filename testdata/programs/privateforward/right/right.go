package right

type I interface{ get() int }

func Implements(v any) bool { _, ok := v.(I); return ok }
