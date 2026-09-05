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

func TestStripJoinsAnEmptyBody(t *testing.T) {
	in := "void f() {\n" +
		"  //#fmain.go|main.f\n" +
		"}\n" +
		"\n" +
		"void g() {\n" +
		"  //#fmain.go|main.g\n" +
		"  //#g7\n" +
		"  a();\n" +
		"}\n"
	text, data := Strip(in)
	want := "void f() {}\n\nvoid g() {\n  a();\n}\n"
	if text != want {
		t.Errorf("got the text %q, want %q", text, want)
	}
	// f stands on the line 1, and the body of g starts on the line 4.
	if wantData := "1|main.go|main.f;3|main.go|main.g\n4+7"; data != wantData {
		t.Errorf("got the map %q, want %q", data, wantData)
	}
}

func TestStripKeepsAnElseBlock(t *testing.T) {
	in := "void f() {\n" +
		"  //#fmain.go|main.f\n" +
		"  //#g7\n" +
		"  if (c) {\n" +
		"    //#g8\n" +
		"  } else {\n" +
		"    a();\n" +
		"  }\n" +
		"}\n"
	text, _ := Strip(in)
	want := "void f() {\n  if (c) {\n  } else {\n    a();\n  }\n}\n"
	if text != want {
		t.Errorf("got the text %q, want %q", text, want)
	}
}
