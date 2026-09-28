package analyze

// Recursive calls need resumable frames even when they do not wait for I/O.
func (a *analyzer) recursive() {
	index := map[any]int{}
	low := map[any]int{}
	active := map[any]bool{}
	var stack []any
	next := 0
	var visit func(any)
	visit = func(key any) {
		next++
		index[key], low[key] = next, next
		active[key] = true
		stack = append(stack, key)
		self := false
		for _, site := range a.summaries[key].sites {
			for _, target := range site.targets {
				target = blockTarget(target)
				if a.summaries[target] == nil {
					continue
				}
				self = self || target == key
				if index[target] == 0 {
					visit(target)
					low[key] = min(low[key], low[target])
				} else if active[target] {
					low[key] = min(low[key], index[target])
				}
			}
		}
		if low[key] != index[key] {
			return
		}
		var component []any
		for {
			member := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			active[member] = false
			component = append(component, member)
			if member == key {
				break
			}
		}
		if len(component) > 1 || self {
			for _, member := range component {
				a.result.blocking[member] = true
			}
		}
	}
	for _, key := range a.order {
		if index[key] == 0 {
			visit(key)
		}
	}
}
