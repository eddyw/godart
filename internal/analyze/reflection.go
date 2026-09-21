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
