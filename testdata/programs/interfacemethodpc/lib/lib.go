package lib

type Alias = int
type Box int

func (b Box) hidden() int { return int(b) }

type Named interface{ hidden() int }

func Values() []any {
	var value interface{ hidden() int } = Box(1)
	return []any{Named.hidden, (interface{ hidden() int }).hidden, value.hidden,
		(interface{ Shape(struct{ hidden int }) }).Shape,
		(interface{ AliasShape(struct{ Alias }) }).AliasShape,
		(interface{ BasicShape(struct{ int }) }).BasicShape,
		(interface{ AnyShape(struct{ any }) }).AnyShape,
	}
}
