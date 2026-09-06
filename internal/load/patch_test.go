package load

import (
	"strings"
	"testing"
)

const patchSource = `package p

import (
	"testing"
	"unsafe"
)

func TestA(t *testing.T) {
	a := "x"
	if unsafe.StringData(a) == nil {
		t.Fatal("nil")
	}
}
`

// TestApplyPatchChangesTheNamedLines proves that a hunk removes and adds lines
// at the line of its header, and that the text before the first header is free.
func TestApplyPatchChangesTheNamedLines(t *testing.T) {
	patch := `The test reads the address of a string, and the subset holds no such read.
--- a_test.go
+++ a_test.go
@@ -3,6 +3,5 @@
 import (
 	"testing"
-	"unsafe"
 )
 
 func TestA(t *testing.T) {
@@ -9,4 +8,3 @@ func TestA(t *testing.T) {
 	a := "x"
-	if unsafe.StringData(a) == nil {
-		t.Fatal("nil")
-	}
+	_ = a
 }
`
	got, err := applyPatch([]byte(patchSource), []byte(patch), "a_test.go.patch")
	if err != nil {
		t.Fatal(err)
	}
	want := `package p

import (
	"testing"
)

func TestA(t *testing.T) {
	a := "x"
	_ = a
}
`
	if string(got) != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

// TestApplyPatchRejectsADrift proves that a context line that differs from the
// file stops the build with the line.
func TestApplyPatchRejectsADrift(t *testing.T) {
	patch := `@@ -9,2 +9,1 @@
 	a := "y"
-	if unsafe.StringData(a) == nil {
`
	_, err := applyPatch([]byte(patchSource), []byte(patch), "a_test.go.patch")
	if err == nil || !strings.Contains(err.Error(), "line 9 of the file differs") {
		t.Fatalf("got %v, want a report of line 9", err)
	}
}

// TestApplyPatchKeepsALastLineWithNoNewline proves that a file whose last line
// holds no newline keeps that form.
func TestApplyPatchKeepsALastLineWithNoNewline(t *testing.T) {
	src := "a\nb\nc"
	patch := "@@ -1,1 +1,1 @@\n-a\n+A\n"
	got, err := applyPatch([]byte(src), []byte(patch), "x.patch")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "A\nb\nc" {
		t.Errorf("got %q, want %q", got, "A\nb\nc")
	}
}
