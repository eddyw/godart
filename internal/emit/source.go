package emit

import (
	"fmt"
	"go/token"
	"go/types"
)

func (e *emitter) openSourceFrame() func() {
	previous := e.sourceFrame
	e.sourceFrame = e.cfg.Fset != nil && e.curGoName != ""
	if !e.sourceFrame {
		return func() { e.sourceFrame = previous }
	}
	e.line("final $source = $sourceEnter(%s, %s, %d);", dartString(e.curGoName), dartString(e.funcFile), e.funcLine)
	e.line("try {")
	return func() {
		e.line("} finally { $sourceLeave($source); }")
		e.sourceFrame = previous
	}
}

func (e *emitter) sourcePosition(pos token.Pos) {
	if !e.sourceFrame || !pos.IsValid() {
		return
	}
	e.line("$source?.at(%s, %d);", dartString(e.goFile(pos)), e.cfg.Fset.Position(pos).Line)
}

func (e *emitter) sourceCall(pos token.Pos, text string, typ types.Type) string {
	if !e.sourceFrame || !pos.IsValid() {
		return text
	}
	result := e.dartType(typ)
	if tuple, ok := typ.(*types.Tuple); ok && tuple.Len() == 0 {
		result = "void"
	}
	// Each call in a compound initializer needs its own source position.
	// The helper closure has no Go frame. Its source region stays unmapped.
	line, file := e.cfg.Fset.Position(pos).Line, e.goFile(pos)
	return fmt.Sprintf("\n%s%d|%s\n$sourceCall<%s>($source, %s, %d,\n%s0\n() => %s\n%s%d|%s\n)", lineMark, line, file, result, dartString(file), line, lineMark, text, lineMark, line, file)
}
