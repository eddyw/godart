package analyze

import "go/ast"

type reflectedFunction struct{ call *ast.CallExpr }

func (a *analyzer) reflectFactory(call *ast.CallExpr) {
	key := reflectedFunction{call}
	s := &summary{key: key}
	site := &callSite{summary: s, callee: a.flowExpr(call.Args[1]), done: map[any]bool{}}
	s.sites = []*callSite{site}
	a.summaries[key] = s
	a.order = append(a.order, key)
	a.sites = append(a.sites, site)
	a.reflected = append(a.reflected, key)
}

func (a *analyzer) reflectEscapes() {
	if len(a.reflected) == 0 {
		return
	}
	groups := map[sigCell]bool{}
	collect := func(c cell) {
		if g, ok := c.(sigCell); ok {
			groups[g] = true
		}
	}
	for c := range a.pts {
		collect(c)
	}
	for c, edges := range a.edges {
		collect(c)
		for dst := range edges {
			collect(dst)
		}
	}
	for _, site := range a.sites {
		for _, group := range site.resultEsc {
			groups[sigCell{group}] = true
		}
		for _, c := range site.callee.cells {
			collect(c)
		}
	}
	// Reflection can replace an escaped function with a synthetic callback.
	for group := range groups {
		for _, key := range a.reflected {
			a.addTarget(group, key)
		}
	}
}

func (a *analyzer) reflectCalls(c cell) {
	if !a.reflection {
		return
	}
	if _, ok := c.(sigCell); !ok {
		return
	}
	// A reflection call has no Go call site for argument or result flow.
	for key := range a.pts[c] {
		sig := a.sigOf(blockTarget(key))
		if sig == nil {
			continue
		}
		for i := 0; i < sig.Results().Len(); i++ {
			result := funcSig(sig.Results().At(i).Type())
			if result == nil {
				continue
			}
			name := signatureKey(result)
			groups := a.byGroup[a.find(name)]
			if len(groups) == 0 {
				groups = []string{name}
			}
			for _, group := range groups {
				a.addEdge(retCell{blockTarget(key), i}, sigCell{group})
			}
		}

		for i := 0; i < sig.Params().Len(); i++ {
			param := sig.Params().At(i)
			callback := funcSig(param.Type())
			if callback == nil {
				continue
			}
			key := signatureKey(callback)
			groups := a.byGroup[a.find(key)]
			if len(groups) == 0 {
				groups = []string{key}
			}
			for _, name := range groups {
				group := sigCell{name}
				for _, factory := range a.reflected {
					a.addTarget(group, factory)
				}
				a.addEdge(group, funcCell(param))
			}
		}
	}
}
