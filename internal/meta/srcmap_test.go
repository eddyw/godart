package meta

import "testing"

func TestStripRemovesTheMarkers(t *testing.T) {
	in := "void f() {\n" +
		"  //#fmain.go|main.f\n" +
		"  //#g12\n" +
		"  a();\n" +
		"  //#g13\n" +
		"  b();\n" +
		"}\n"
	text, data := Strip(in)
	want := "void f() {\n  a();\n  b();\n}\n"
	if text != want {
		t.Errorf("got the text %q, want %q", text, want)
	}
	// The function starts at line 2, and the 2 lines follow it.
	if wantData := "2|main.go|main.f\n2+c,1+1"; data != wantData {
		t.Errorf("got the map %q, want %q", data, wantData)
	}
}

func TestStripKeepsAFileWithNoMarker(t *testing.T) {
	in := "void f() {\n  a();\n}\n"
	text, data := Strip(in)
	if text != in {
		t.Errorf("got the text %q, want %q", text, in)
	}
	if data != "\n" {
		t.Errorf("got the map %q, want an empty map", data)
	}
}
