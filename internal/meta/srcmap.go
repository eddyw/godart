package meta

import (
	"strconv"
	"strings"
)

// The 2 markers that stage 5 writes. Read the constants of the package emit.
const (
	fileMark = "//#f"
	lineMark = "//#g"
)

// Strip removes the markers of the source map from 1 Dart file. It gives the
// text of the file and the map of the file.
//
// dart format runs in front of this step, thus the line numbers of the map name
// the lines that the reader of a stack trace sees. A marker stands on a line of
// its own, and the removal of such a line moves each line after it by 1.
func Strip(text string) (string, string) {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	var funcs, entries []string
	var pendingFile, pendingFunc, pendingLineFile string
	pendingLine := 0
	pendingSet := false
	lastFunc, lastDart, lastGo := 0, 0, 0
	dropped := false
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		switch {
		case strings.HasPrefix(trimmed, fileMark):
			rest := trimmed[len(fileMark):]
			file, fn, _ := strings.Cut(rest, "|")
			pendingFile, pendingFunc = file, fn
			dropped = true
			continue
		case strings.HasPrefix(trimmed, lineMark):
			rest := trimmed[len(lineMark):]
			// A marker names its file only when a line directive changed it.
			// The line 0 unmaps the output line: the reader of the map drops
			// such a frame from a stack.
			numText, file, _ := strings.Cut(rest, "|")
			n, err := strconv.Atoi(numText)
			if err == nil {
				pendingLine = n
				pendingLineFile = file
				pendingSet = true
			}
			dropped = true
			continue
		}
		// A marker that stood alone in a body leaves the body empty. dart
		// format writes an empty body on 1 line, thus the 2 lines join again
		// and the output keeps the form that dart format gives.
		if dropped && trimmed == "}" && len(out) > 0 &&
			strings.HasSuffix(out[len(out)-1], "{") {
			out[len(out)-1] += "}"
		} else {
			out = append(out, line)
		}
		dropped = false
		at := len(out)
		if pendingFunc != "" {
			funcs = append(funcs, base36(at-lastFunc)+"|"+pendingFile+"|"+pendingFunc)
			lastFunc = at
			pendingFile, pendingFunc = "", ""
		}
		if pendingSet {
			entry := base36(at-lastDart) + delta(pendingLine-lastGo)
			if pendingLineFile != "" {
				entry += "|" + pendingLineFile
			}
			entries = append(entries, entry)
			lastDart, lastGo = at, pendingLine
			pendingSet = false
			pendingLineFile = ""
		}
	}
	return strings.Join(out, "\n"), strings.Join(funcs, ";") + "\n" + strings.Join(entries, ",")
}

func base36(n int) string { return strconv.FormatInt(int64(n), 36) }

func delta(n int) string {
	if n < 0 {
		return "-" + base36(-n)
	}
	return "+" + base36(n)
}
