package main

import (
	"context"
	"fmt"
	"log/slog"
	"testing/slogtest"
)

// The program builds the result maps itself. The example of Go reads
// encoding/json for that step, and docs/parity.md holds the reason that the
// program takes another way.
func main() {
	var lines []map[string]any
	h := &recorder{sink: &lines}

	err := slogtest.TestHandler(h, func() []map[string]any { return lines })
	fmt.Println("recorder passes:", err == nil)
	if err != nil {
		fmt.Println("cause:", err)
	}

	// A handler that forgets each group breaks the rules of the suite.
	var broken []map[string]any
	b := &recorder{sink: &broken, flat: true}
	err = slogtest.TestHandler(b, func() []map[string]any { return broken })
	fmt.Println("flat handler fails:", err != nil)
}

// recorder is a slog.Handler that writes each record as a map. A group holds a
// map of its own, exactly as the suite asks.
type recorder struct {
	sink   *[]map[string]any
	parent *recorder
	group  string
	attrs  []slog.Attr
	flat   bool
}

func (r *recorder) Enabled(_ context.Context, level slog.Level) bool {
	return level >= slog.LevelInfo
}

func (r *recorder) WithAttrs(as []slog.Attr) slog.Handler {
	out := *r
	out.attrs = append(append([]slog.Attr{}, r.attrs...), as...)
	return &out
}

func (r *recorder) WithGroup(name string) slog.Handler {
	if name == "" || r.flat {
		return r
	}
	return &recorder{sink: r.sink, parent: r, group: name, flat: r.flat}
}

func (r *recorder) Handle(_ context.Context, rec slog.Record) error {
	cur := map[string]any{}
	rec.Attrs(func(a slog.Attr) bool {
		add(cur, a, r.flat)
		return true
	})
	for node := r; node != nil; node = node.parent {
		level := map[string]any{}
		for _, a := range node.attrs {
			add(level, a, r.flat)
		}
		for k, v := range cur {
			level[k] = v
		}
		if node.group == "" {
			cur = level
			continue
		}
		outer := map[string]any{}
		if len(level) > 0 {
			outer[node.group] = level
		}
		cur = outer
	}
	if !rec.Time.IsZero() {
		cur[slog.TimeKey] = rec.Time
	}
	cur[slog.LevelKey] = rec.Level
	cur[slog.MessageKey] = rec.Message
	*r.sink = append(*r.sink, cur)
	return nil
}

// add writes 1 attribute into a map. It resolves the value first, it drops an
// empty attribute and an empty group, and it inlines a group with no key.
func add(m map[string]any, a slog.Attr, flat bool) {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return
	}
	if a.Value.Kind() != slog.KindGroup {
		m[a.Key] = a.Value.Any()
		return
	}
	group := a.Value.Group()
	if len(group) == 0 {
		return
	}
	if a.Key == "" || flat {
		for _, one := range group {
			add(m, one, flat)
		}
		return
	}
	inner := map[string]any{}
	for _, one := range group {
		add(inner, one, flat)
	}
	if len(inner) > 0 {
		m[a.Key] = inner
	}
}
