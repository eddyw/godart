// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE-Go file.

package load

import (
	"errors"
	"fmt"
	"go/token"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

type diagnostics struct {
	errors []error
	limit  bool
}

func (d *diagnostics) add(err error) {
	if e, ok := err.(packages.Error); ok && strings.HasPrefix(e.Msg, "\t") && len(d.errors) != 0 {
		i := len(d.errors) - 1
		d.errors[i] = fmt.Errorf("%w\n\t%s: %s", d.errors[i], e.Pos, strings.TrimPrefix(e.Msg, "\t"))
	} else {
		d.errors = append(d.errors, err)
	}
}

func (d *diagnostics) err() error {
	var selected []error
	var last token.Position
	var lastMessage string
	for _, err := range d.errors {
		pos := diagnosticPosition(err)
		_, message, _ := strings.Cut(err.Error(), ": ")
		if pos.IsValid() && pos.Filename == last.Filename && pos.Line == last.Line && message == lastMessage {
			continue
		}
		selected = append(selected, err)
		last, lastMessage = pos, message
		if d.limit && len(selected) == 10 {
			break
		}
	}
	sortDiagnostics(selected)
	if d.limit && len(selected) == 10 {
		selected = append(selected, fmt.Errorf("%s: too many errors", last))
	}
	return errors.Join(selected...)
}

func sortDiagnostics(errs []error) {
	sort.SliceStable(errs, func(i, j int) bool {
		a, b := diagnosticPosition(errs[i]), diagnosticPosition(errs[j])
		if a.Filename != b.Filename {
			return a.Filename < b.Filename
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Column < b.Column
	})
}

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
