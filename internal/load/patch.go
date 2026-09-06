package load

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// applyPatch applies a unified diff to the text of a file. Each hunk must match
// the text at the line that its header names, and a mismatch stops the build
// with the line. A test file of Go changes with each release, thus a diff that
// applies at another place could change a test in silence.
//
// The text before the first hunk header is free, thus a diff can carry the
// cause of each change there.
func applyPatch(src, patch []byte, name string) ([]byte, error) {
	lines := splitLines(src)
	var out bytes.Buffer
	next := 0
	hunks := strings.Split(string(patch), "\n")
	for i := 0; i < len(hunks); i++ {
		if !strings.HasPrefix(hunks[i], "@@ ") {
			continue
		}
		start, err := hunkStart(hunks[i])
		if err != nil {
			return nil, fmt.Errorf("%s: %v", name, err)
		}
		if start < next || start > len(lines) {
			return nil, fmt.Errorf("%s: the hunk %q names line %d, and the file holds %d lines",
				name, hunks[i], start+1, len(lines))
		}
		for _, l := range lines[next:start] {
			out.WriteString(l)
		}
		next = start
		for i+1 < len(hunks) && !strings.HasPrefix(hunks[i+1], "@@ ") {
			i++
			l := hunks[i]
			if l == "" || strings.HasPrefix(l, "\\") {
				continue
			}
			switch l[0] {
			case ' ', '-':
				if next >= len(lines) || strings.TrimRight(lines[next], "\n") != l[1:] {
					return nil, fmt.Errorf("%s: line %d of the file differs from the diff: %q",
						name, next+1, l[1:])
				}
				if l[0] == ' ' {
					out.WriteString(lines[next])
				}
				next++
			case '+':
				out.WriteString(l[1:])
				out.WriteByte('\n')
			default:
				return nil, fmt.Errorf("%s: the line %q of the diff starts with no space, plus, or minus",
					name, l)
			}
		}
	}
	for _, l := range lines[next:] {
		out.WriteString(l)
	}
	return out.Bytes(), nil
}

// hunkStart reads the line of the file that a hunk header names, from 0.
func hunkStart(header string) (int, error) {
	fields := strings.Fields(header)
	if len(fields) < 3 || !strings.HasPrefix(fields[1], "-") {
		return 0, fmt.Errorf("the hunk header %q holds no line", header)
	}
	text, _, _ := strings.Cut(fields[1][1:], ",")
	n, err := strconv.Atoi(text)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("the hunk header %q holds no line", header)
	}
	return n - 1, nil
}

// splitLines gives each line with its newline. A last line with no newline
// stands alone.
func splitLines(src []byte) []string {
	var out []string
	for len(src) > 0 {
		i := bytes.IndexByte(src, '\n')
		if i < 0 {
			out = append(out, string(src))
			break
		}
		out = append(out, string(src[:i+1]))
		src = src[i+1:]
	}
	return out
}
