package bench

import (
	"testing"
	"unicode/utf8"

	"gart.dev/ui"
)

// The measurement reads the length of each literal as a fact, thus a drift of
// 1 character would make the report wrong.
func TestLiteralLength(t *testing.T) {
	cases := []struct {
		name  string
		text  string
		runes int
		bytes int
	}{
		{"lit5", lit5, 5, 5},
		{"lit20", lit20, 20, 20},
		{"lit80", lit80, 80, 80},
		{"litUTF8", litUTF8, 20, 37},
	}
	for _, c := range cases {
		if got := utf8.RuneCountInString(c.text); got != c.runes {
			t.Errorf("%s holds %d runes, and the report says %d", c.name, got, c.runes)
		}
		if got := len(c.text); got != c.bytes {
			t.Errorf("%s holds %d bytes, and the report says %d", c.name, got, c.bytes)
		}
	}
}

// The report names a node count, thus the shape of each tree must match it.
func TestTreeShape(t *testing.T) {
	Setup(20, 20, false)
	if got := Nodes(); got != 21 {
		t.Errorf("Nodes gives %d, and the tree holds 21", got)
	}
	for name, tree := range map[string]ui.Node{
		"held":    TreeHeld(),
		"literal": TreeLiteral(),
		"itoa":    TreeItoa(),
	} {
		col, ok := tree.(ui.Column)
		if !ok {
			t.Fatalf("%s gives %T, and the harness reads a Column", name, tree)
		}
		if len(col.Children) != 20 {
			t.Errorf("%s holds %d children, and the case asks for 20", name, len(col.Children))
		}
	}
}

// A write of 1 signal must mark 1 boundary. The whole measurement of the
// normal case rests on that rule.
func TestOneSignalMarksOneBoundary(t *testing.T) {
	Setup(50, 20, false)
	tree := TreeWatch(false).(ui.Column)
	ids := make([]int, len(tree.Children))
	for i, kid := range tree.Children {
		ids[i] = ui.NewWatch()
		ui.RunWatch(ids[i], kid.(ui.Watch).Build)
	}
	ui.TakeDirty()
	Bump(7)
	dirty := Dirty()
	if len(dirty) != 1 {
		t.Fatalf("a write of 1 signal marked %d boundaries", len(dirty))
	}
	if dirty[0] != ids[7] {
		t.Errorf("the dirty boundary is %d, and row 7 holds %d", dirty[0], ids[7])
	}
}

// The diff measurement counts on 2 text pools of the same length that differ
// in the last byte alone.
func TestPickPools(t *testing.T) {
	Setup(30, 20, false)
	col := TreePick(0).(ui.Column)
	other := TreePick(30).(ui.Column)
	for i := range col.Children {
		a := col.Children[i].(ui.Text).Data
		b := other.Children[i].(ui.Text).Data
		if len(a) != 20 || len(b) != 20 {
			t.Fatalf("row %d holds %d and %d bytes, and the case asks for 20", i, len(a), len(b))
		}
		if a == b {
			t.Fatalf("row %d holds the same text in the 2 pools", i)
		}
		if a[:19] != b[:19] {
			t.Errorf("row %d differs before the last byte", i)
		}
	}
	one := TreePick(1).(ui.Column)
	diff := 0
	for i := range col.Children {
		if col.Children[i].(ui.Text).Data != one.Children[i].(ui.Text).Data {
			diff++
		}
	}
	if diff != 1 {
		t.Errorf("TreePick(1) differs in %d rows, and the case asks for 1", diff)
	}
}

// The styled tree must set 1 optional field, because that field proves the
// trap of a diff that reads the generated ==.
func TestPickStyled(t *testing.T) {
	Setup(5, 20, false)
	col := TreePickStyled(0).(ui.Column)
	for i, kid := range col.Children {
		if kid.(ui.Text).TextAlign == nil {
			t.Errorf("row %d sets no TextAlign", i)
		}
	}
}
