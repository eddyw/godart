package meta_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/meta"
)

func TestStoreLongPackagePaths(t *testing.T) {
	store := meta.Open(t.TempDir())
	prefix := "_/" + strings.Repeat("source/", 60)
	paths := []string{"example.com/package", prefix + "a [" + prefix + "a.test]", prefix + "b [" + prefix + "b.test]"}
	archives := make([]*meta.Archive, len(paths))
	for i, path := range paths {
		archives[i] = &meta.Archive{Path: path, Name: "p", SourceHash: path, Files: []string{"p.dart"}}
		if err := store.Write(archives[i]); err != nil {
			t.Fatal(err)
		}
	}
	for _, want := range archives {
		got, ok := store.Read(want.Path)
		if !ok || !reflect.DeepEqual(got, want) {
			t.Fatalf("archive: got %+v, %t; want %+v", got, ok, want)
		}
	}
}
