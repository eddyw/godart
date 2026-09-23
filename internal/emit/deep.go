package emit

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/eddyw/godart/internal/ir"
)

const deepSliceLevels = 32

// Separate helpers bound Dart parser depth without a change to operand order.
func (e *emitter) deepSlice(x *ir.Composite) string {
	var chain []*ir.Composite
	for {
		if _, ok := x.Type.Underlying().(*types.Slice); !ok || x.Results != nil || x.Indices != nil || len(x.Elems) != 1 {
			break
		}
		chain = append(chain, x)
		inner, ok := x.Elems[0].(*ir.Composite)
		if !ok {
			break
		}
		x = inner
	}
	if len(chain) < deepSliceLevels {
		return ""
	}
	text := e.composite(chain[len(chain)-1])
	for count := len(chain) - 1; count > 0; {
		n := min(count, 16)
		text = e.sliceWrapper(n) + "(" + text + ")"
		count -= n
	}
	return text
}

func (e *emitter) sliceWrapper(n int) string {
	if e.lits.sliceWraps == nil {
		e.lits.sliceWraps = map[int]string{}
	}
	if name := e.lits.sliceWraps[n]; name != "" {
		return name
	}
	name := fmt.Sprintf("$sliceWrap%d_%d", e.lits.tag, n)
	e.lits.sliceWraps[n] = name
	result := strings.Repeat("Slice<", n) + "T" + strings.Repeat(">", n)
	var body strings.Builder
	previous := "value"
	for i := 0; i < n; i++ {
		next := fmt.Sprintf("v%d", i)
		fmt.Fprintf(&body, "final %s = Slice.of([%s]);\n", next, previous)
		previous = next
	}
	fmt.Fprintf(&body, "return %s;", previous)
	e.lits.decls = append(e.lits.decls, fmt.Sprintf("%s %s<T>(T value) {\n%s\n}", result, name, body.String()))
	return name
}

// Constant indexes permit a helper without an earlier read of another operand.
func (e *emitter) deepIndex(x *ir.Index) string {
	var chain []*ir.Index
	var base ir.Expr = x
	for {
		next, ok := base.(*ir.Index)
		if !ok {
			break
		}
		if _, ok := next.X.GoType().Underlying().(*types.Slice); !ok {
			break
		}
		if _, ok := next.Index.(*ir.Const); !ok || e.unsignedBound(next.Index) != "false" {
			break
		}
		chain = append(chain, next)
		base = next.X
	}
	if len(chain) < 16 {
		return ""
	}
	if e.lits.sliceRead == "" {
		e.lits.sliceRead = fmt.Sprintf("$sliceRead%d", e.lits.tag)
		e.lits.decls = append(e.lits.decls, fmt.Sprintf(`T %s<T>(Object? source, List<GoInt> indexes) {
    Object? value = source;
    for (final index in indexes) {
      value = (value as Slice).$at(index);
    }
    return value as T;
  }`, e.lits.sliceRead))
	}
	indexes := make([]string, len(chain))
	for i, node := range chain {
		indexes[len(chain)-1-i] = e.expr(node.Index)
	}
	return fmt.Sprintf("%s<%s>(%s, <GoInt>[%s])", e.lits.sliceRead, e.dartType(x.Type), e.expr(base), strings.Join(indexes, ", "))
}
