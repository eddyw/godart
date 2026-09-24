package analyze

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"strings"
)

// A synchronous function type must return its final result to a Dart caller.
// Reject actual waits before recursive calls receive resumable frames.
// Value-flow analysis includes indirect callbacks and reflective assignments.

const syncDirective = "//gart:sync"

// SyncError reports 1 rejection of the synchronous check.
type SyncError struct {
	Pos  token.Position
	Msg  string
	Hint string
}

func (e *SyncError) Error() string {
	if e.Hint == "" {
		return fmt.Sprintf("%s: %s", e.Pos, e.Msg)
	}
	return fmt.Sprintf("%s: %s (%s)", e.Pos, e.Msg, e.Hint)
}

// SyncErrors joins each rejection of the synchronous check, in source order.
func (r *Result) SyncErrors() error {
	if len(r.syncErrs) == 0 {
		return nil
	}
	out := make([]error, len(r.syncErrs))
	for i, e := range r.syncErrs {
		out[i] = e
	}
	return errors.Join(out...)
}

// typeSpec reads 1 type declaration and keeps a synchronous function type.
// The doc comment of a lone declaration stands on the declaration, and the doc
// comment of a member of a group stands on the member.
func (a *analyzer) typeSpec(d *ast.GenDecl, spec *ast.TypeSpec) {
	doc := spec.Doc
	if doc == nil && len(d.Specs) == 1 {
		doc = d.Doc
	}
	if !hasDirective(doc) {
		return
	}
	obj, _ := a.pkg.TypesInfo.Defs[spec.Name].(*types.TypeName)
	if obj == nil {
		return
	}
	if funcSig(obj.Type()) == nil {
		a.syncError(spec.Pos(),
			"the directive "+syncDirective+" stands on a type that is not a function type",
			"only a named function type names a synchronous position")
		return
	}
	if a.syncTypes == nil {
		a.syncTypes = map[*types.TypeName]bool{}
	}
	a.syncTypes[obj] = true
}

func hasDirective(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}
	for _, c := range doc.List {
		if strings.TrimSpace(c.Text) == syncDirective {
			return true
		}
	}
	return false
}

func (a *analyzer) syncError(pos token.Pos, msg, hint string) {
	a.result.syncErrs = append(a.result.syncErrs, &SyncError{
		Pos:  a.fset.Position(pos),
		Msg:  msg,
		Hint: hint,
	})
}

// checkSync rejects each function that can block in a synchronous position.
// It follows the wait analysis and precedes recursive frame propagation.
func (a *analyzer) checkSync() {
	if len(a.syncTypes) > 0 {
		for c, set := range a.pts {
			v, ok := c.(*types.Var)
			if !ok {
				continue
			}
			name := a.syncName(v.Type())
			if name == "" {
				continue
			}
			for key := range set {
				a.reportSync(v, name, blockTarget(key))
			}
		}
	}
	slices.SortFunc(a.result.syncErrs, func(x, y *SyncError) int {
		return strings.Compare(x.Error(), y.Error())
	})
}

// syncName gives the name of a synchronous function type, or an empty text.
func (a *analyzer) syncName(t types.Type) string {
	named, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return ""
	}
	obj := named.Origin().Obj()
	if !a.syncTypes[obj] {
		return ""
	}
	if obj.Pkg() == nil {
		return obj.Name()
	}
	return obj.Pkg().Name() + "." + obj.Name()
}

func (a *analyzer) reportSync(pos *types.Var, typeName string, key any) {
	if !a.result.blocking[key] {
		return
	}
	valuePos := keyPos(key)
	if !valuePos.IsValid() {
		return
	}
	hint := pos.Name() + " at " + a.fset.Position(pos.Pos()).String() + " holds the value"
	if cause, op := a.blockCause(key); cause.IsValid() {
		hint += ", and " + op + " at " + a.fset.Position(cause).String() + " blocks"
	}
	a.syncError(valuePos,
		"this function can block, and "+typeName+" is a synchronous position",
		hint)
}

func keyPos(key any) token.Pos {
	switch key := key.(type) {
	case *types.Func:
		return key.Pos()
	case *ast.FuncLit:
		return key.Pos()
	case *ast.RangeStmt:
		return key.Pos()
	case reflectedFunction:
		return key.call.Pos()
	}
	return token.NoPos
}

// blockCause follows the call chain of a function that blocks until it reaches
// the operation of a body that blocks.
func (a *analyzer) blockCause(key any) (token.Pos, string) {
	seen := map[any]bool{}
	for !seen[key] {
		seen[key] = true
		s := a.summaries[key]
		if s == nil {
			return token.NoPos, ""
		}
		if s.direct {
			return s.directPos, s.directWhat
		}
		next, pos := a.blockingCall(s)
		if next == nil {
			return pos, "a call through a function value"
		}
		key = next
	}
	return token.NoPos, ""
}

// blockingCall gives the first call site of a body that waits, and the target
// of it that blocks. The site list stands in source order, and the target with
// the lowest position wins, thus 2 runs give the same answer.
func (a *analyzer) blockingCall(s *summary) (any, token.Pos) {
	for _, site := range s.sites {
		if !a.siteAsync(site) {
			continue
		}
		var best any
		for _, t := range site.targets {
			if !a.result.blocking[t] {
				continue
			}
			if best == nil || keyPos(t) < keyPos(best) {
				best = t
			}
		}
		return best, sitePos(site)
	}
	return nil, token.NoPos
}

func sitePos(site *callSite) token.Pos {
	if site.call != nil {
		return site.call.Pos()
	}
	if site.rng != nil {
		return site.rng.Pos()
	}
	return token.NoPos
}

func (r *Result) SyncType(t types.Type) bool {
	named, ok := types.Unalias(t).(*types.Named)
	return ok && r.syncTypes[named.Origin().Obj()]
}

func (r *Result) SyncTypes() map[*types.TypeName]bool { return r.syncTypes }
