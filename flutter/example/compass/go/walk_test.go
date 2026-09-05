package app

import (
	"reflect"
	"strings"

	"gart.dev/ui"
)

// textOf walks 1 node tree and it gives each text of it, in the order of the
// tree. A Watch node holds a closure, thus the walk calls it and it walks the
// result.
func textOf(n any) []string {
	var out []string
	walk(reflect.ValueOf(n), &out, 0)
	return out
}

// text gives each text of a tree in 1 line.
func text(n any) string { return strings.Join(textOf(n), " | ") }

func walk(v reflect.Value, out *[]string, depth int) {
	if depth > 60 || !v.IsValid() {
		return
	}
	switch v.Kind() {
	case reflect.Interface, reflect.Pointer:
		if !v.IsNil() {
			walk(v.Elem(), out, depth+1)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			walk(v.Index(i), out, depth+1)
		}
	case reflect.Struct:
		if t, ok := v.Interface().(ui.Text); ok {
			*out = append(*out, t.Data)
			return
		}
		if w, ok := v.Interface().(ui.Watch); ok {
			if w.Build != nil {
				walk(reflect.ValueOf(w.Build()), out, depth+1)
			}
			return
		}
		if in, ok := v.Interface().(ui.TextInput); ok {
			*out = append(*out, "["+in.Label+"="+in.Value+"]")
			return
		}
		if ic, ok := v.Interface().(ui.Icon); ok {
			*out = append(*out, "<"+ic.Icon+">")
			return
		}
		for i := 0; i < v.NumField(); i++ {
			walk(v.Field(i), out, depth+1)
		}
	}
}

// tap runs the first callback of the tree whose text matches name. It reports
// whether it found 1.
func tap(n any, name string) bool {
	return findTap(reflect.ValueOf(n), name, 0)
}

func findTap(v reflect.Value, name string, depth int) bool {
	if depth > 60 || !v.IsValid() {
		return false
	}
	switch v.Kind() {
	case reflect.Interface, reflect.Pointer:
		return !v.IsNil() && findTap(v.Elem(), name, depth+1)
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if findTap(v.Index(i), name, depth+1) {
				return true
			}
		}
	case reflect.Struct:
		if w, ok := v.Interface().(ui.Watch); ok {
			return w.Build != nil && findTap(reflect.ValueOf(w.Build()), name, depth+1)
		}
		if label(v) == name {
			if fn := action(v); fn != nil {
				fn()
				return true
			}
		}
		for i := 0; i < v.NumField(); i++ {
			if findTap(v.Field(i), name, depth+1) {
				return true
			}
		}
	}
	return false
}

// label gives the text of a widget that holds 1 child text, or the name of the
// icon of a button.
func label(v reflect.Value) string {
	names := []string{"Child", "Label", "Title", "Icon"}
	for _, n := range names {
		f := v.FieldByName(n)
		if !f.IsValid() {
			continue
		}
		if !f.CanInterface() {
			continue
		}
		if f.Kind() == reflect.Interface && f.IsNil() {
			continue
		}
		switch c := f.Interface().(type) {
		case ui.Text:
			return c.Data
		case ui.Icon:
			return c.Icon
		}
		if f.Kind() == reflect.Interface {
			if t, ok := f.Elem().Interface().(ui.Text); ok {
				return t.Data
			}
			if ic, ok := f.Elem().Interface().(ui.Icon); ok {
				return ic.Icon
			}
		}
	}
	return ""
}

// action gives the callback of a widget that holds 1.
func action(v reflect.Value) func() {
	for _, n := range []string{"OnPressed", "OnTap"} {
		f := v.FieldByName(n)
		if f.IsValid() && f.Kind() == reflect.Func && !f.IsNil() {
			if fn, ok := f.Interface().(func()); ok {
				return fn
			}
		}
	}
	return nil
}
