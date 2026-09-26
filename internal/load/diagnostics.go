package load

import (
	"go/token"
	"strconv"
	"strings"
)

func diagnosticPosition(err error) token.Position {
	text, _, _ := strings.Cut(err.Error(), ": ")
	pos := token.Position{Filename: text}
	last := strings.LastIndexByte(text, ':')
	if last < 0 {
		return pos
	}
	value, parseErr := strconv.Atoi(text[last+1:])
	if parseErr != nil {
		return pos
	}
	pos.Filename, pos.Line = text[:last], value
	last = strings.LastIndexByte(pos.Filename, ':')
	if last >= 0 {
		if line, err := strconv.Atoi(pos.Filename[last+1:]); err == nil {
			pos.Filename, pos.Line, pos.Column = pos.Filename[:last], line, value
		}
	}
	return pos
}
