package analyze

// Recursive calls need resumable frames even when they do not wait for I/O.
func (a *analyzer) recursive() {
	state := map[any]int{}
	index := map[any]int{}
	var stack []any
	var visit func(any)
	visit = func(key any) {
		if state[key] == 2 {
			return
		}
		if state[key] == 1 {
			for _, member := range stack[index[key]:] {
				a.result.blocking[member] = true
			}
			return
		}
		summary := a.summaries[key]
		if summary == nil {
			return
		}
		state[key] = 1
		index[key] = len(stack)
		stack = append(stack, key)
		for _, site := range summary.sites {
			for _, target := range site.targets {
				visit(blockTarget(target))
			}
		}
		stack = stack[:len(stack)-1]
		state[key] = 2
	}
	for _, key := range a.order {
		visit(key)
	}
}
