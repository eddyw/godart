// Package shapes gives an unnamed struct type to another package. Each field
// carries an exported name, thus the 2 packages share 1 Dart class.
package shapes

type Row = struct {
	Name string
	N    int
}

func Make(name string, n int) Row {
	return Row{Name: name, N: n}
}

func Text(r Row) string {
	return r.Name
}

func Empty() struct{} {
	return struct{}{}
}
