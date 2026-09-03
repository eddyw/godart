//go:build dart

package strings

// A Replacer replaces a list of pairs in a string.
type Replacer struct {
	pairs []string
}

// NewReplacer gives a Replacer of the pairs. Each pair holds the old text and
// the new text, and the count of the arguments must be even.
//
// Go matches an empty old text in front of each rune. This Replacer skips such
// a pair. docs/parity.md holds that gap.
func NewReplacer(oldnew ...string) *Replacer {
	if len(oldnew)%2 == 1 {
		panic("strings.NewReplacer: odd argument count")
	}
	return &Replacer{pairs: append([]string(nil), oldnew...)}
}

// Replace gives the text with each pair applied. A match takes the first pair
// of the list that fits, and the walk starts again after the match.
func (r *Replacer) Replace(s string) string {
	var b Builder
	for i := 0; i < len(s); {
		hit := -1
		for k := 0; k+1 < len(r.pairs); k += 2 {
			old := r.pairs[k]
			if old == "" || !HasPrefix(s[i:], old) {
				continue
			}
			hit = k
			break
		}
		if hit < 0 {
			b.WriteByte(s[i])
			i++
			continue
		}
		b.WriteString(r.pairs[hit+1])
		i += len(r.pairs[hit])
	}
	return b.String()
}

// WriteString writes the result of Replace to w.
//
// Go names the type of w io.Writer. The parameter names the same method set,
// and Go holds the 2 types equal.
func (r *Replacer) WriteString(w interface {
	Write(p []byte) (int, error)
}, s string) (int, error) {
	return w.Write([]byte(r.Replace(s)))
}
