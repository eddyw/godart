package normalize

import (
	"go/token"
	"go/types"

	"github.com/eddyw/godart/internal/ir"
)

// The label of the loop that holds the state machine, and the name of the
// variable that holds the number of the next block.
const (
	gotoLabel = "$goto"
	gotoState = "$pc"
)

// stateMachine rewrites a function body that holds a goto statement. The body
// becomes a loop with a switch on a state variable. Each label starts a block,
// and a goto writes the number of its block and starts the loop again.
func (b *builder) stateMachine(body *ir.Block, pos token.Pos) *ir.Block {
	blocks := splitBlocks(body.List)
	index := map[string]int{}
	for i, blk := range blocks {
		if blk.label != "" {
			index[blk.label] = i
		}
	}
	out := &ir.Block{}
	for i := range blocks {
		blocks[i].stmts = hoist(blocks[i].stmts, &out.List)
		blocks[i].stmts = replaceGoto(blocks[i].stmts, index)
	}
	state := &ir.Var{Name: gotoState, Type: types.Typ[types.Int]}
	read := &ir.Ident{Name: gotoState, Type: types.Typ[types.Int]}
	out.List = append(out.List, &ir.Declare{Vars: []*ir.Var{state}, Init: intConst(0)})
	sw := &ir.Switch{Tag: read}
	for i, blk := range blocks {
		sw.Cases = append(sw.Cases, &ir.Case{
			Values:      []ir.Expr{intConst(int64(i))},
			Body:        &ir.Block{List: blk.stmts},
			Fallthrough: i < len(blocks)-1,
		})
	}
	out.List = append(out.List, &ir.For{
		Label: gotoLabel,
		Body:  &ir.Block{List: []ir.Stmt{sw, &ir.Break{Label: gotoLabel}}},
	})
	return out
}

type gotoBlock struct {
	label string
	stmts []ir.Stmt
}

func splitBlocks(list []ir.Stmt) []gotoBlock {
	blocks := []gotoBlock{{}}
	for _, s := range list {
		if l, ok := s.(*ir.Labeled); ok && l.Goto {
			inner, _ := l.Stmt.(*ir.Block)
			blk := gotoBlock{label: l.Label}
			if inner != nil {
				blk.stmts = inner.List
			}
			blocks = append(blocks, blk)
			continue
		}
		blocks[len(blocks)-1].stmts = append(blocks[len(blocks)-1].stmts, s)
	}
	return blocks
}

// hoist moves a declaration out of the state machine. A Dart case clause and
// the next case clause do not share a variable, thus each variable of the top
// level goes in front of the loop.
func hoist(list []ir.Stmt, front *[]ir.Stmt) []ir.Stmt {
	out := make([]ir.Stmt, 0, len(list))
	for _, s := range list {
		d, ok := s.(*ir.Declare)
		if !ok {
			out = append(out, s)
			continue
		}
		*front = append(*front, &ir.Declare{Vars: d.Vars})
		if d.Init == nil {
			continue
		}
		if len(d.Vars) == 1 {
			out = append(out, &ir.Assign{
				Lhs: &ir.Ident{Name: d.Vars[0].Name, Type: d.Vars[0].Type},
				Rhs: d.Init,
			})
			continue
		}
		lhs := make([]ir.Expr, len(d.Vars))
		for i, v := range d.Vars {
			lhs[i] = &ir.Ident{Name: v.Name, Type: v.Type}
		}
		out = append(out, &ir.AssignMulti{Lhs: lhs, Rhs: d.Init})
	}
	return out
}

// replaceGoto writes the block number and starts the loop again.
func replaceGoto(list []ir.Stmt, index map[string]int) []ir.Stmt {
	out := make([]ir.Stmt, 0, len(list))
	for _, s := range list {
		switch s := s.(type) {
		case *ir.Goto:
			n, ok := index[s.Label]
			if !ok {
				out = append(out, s)
				continue
			}
			out = append(out,
				&ir.Assign{
					Lhs: &ir.Ident{Name: gotoState, Type: types.Typ[types.Int]},
					Rhs: intConst(int64(n)),
				},
				&ir.Continue{Label: gotoLabel})
		case *ir.Block:
			s.List = replaceGoto(s.List, index)
			out = append(out, s)
		case *ir.If:
			s.Then.List = replaceGoto(s.Then.List, index)
			if s.Else != nil {
				s.Else = replaceGoto([]ir.Stmt{s.Else}, index)[0]
			}
			out = append(out, s)
		case *ir.For:
			s.Body.List = replaceGoto(s.Body.List, index)
			out = append(out, s)
		case *ir.Switch:
			for _, c := range s.Cases {
				c.Body.List = replaceGoto(c.Body.List, index)
			}
			out = append(out, s)
		case *ir.Labeled:
			s.Stmt = replaceGoto([]ir.Stmt{s.Stmt}, index)[0]
			out = append(out, s)
		default:
			out = append(out, s)
		}
	}
	return out
}
