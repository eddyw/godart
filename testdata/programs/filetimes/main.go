package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"time"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func stamp(path string) (int64, int64) {
	info, err := os.Stat(path)
	must(err)
	stat := reflect.ValueOf(info.Sys()).Elem()
	access := stat.FieldByName("Atim")
	if !access.IsValid() {
		access = stat.FieldByName("Atimespec")
	}
	return access.FieldByName("Sec").Int()*1e9 + access.FieldByName("Nsec").Int(), info.ModTime().UnixNano()
}
func failure(name, path string, a, m time.Time) {
	err := os.Chtimes(path, a, m)
	var detail *os.PathError
	if errors.As(err, &detail) {
		fmt.Println(name, detail.Op, detail.Path == path, errors.Is(err, syscall.ENOENT), errors.Is(err, syscall.ENOTDIR), errors.Is(err, syscall.EINVAL))
		return
	}
	fmt.Println(name, err)
}
func main() {
	root, err := os.MkdirTemp("", "gart-filetimes-")
	must(err)
	defer os.RemoveAll(root)
	file := filepath.Join(root, "file")
	must(os.WriteFile(file, []byte("value"), 0600))
	link := filepath.Join(root, "link")
	must(os.Symlink(file, link))
	for _, path := range []string{file, link, root} {
		for _, n := range []int64{1700000000123456789, -1, -1234567890123456, 0} {
			a, m := time.Unix(0, n), time.Unix(0, n+123)
			must(os.Chtimes(path, a, m))
			gotA, gotM := stamp(path)
			fmt.Println("time", n, gotA, gotM)
			must(os.Chtimes(path, time.Time{}, time.Unix(0, n+456)))
			nextA, nextM := stamp(path)
			fmt.Println("omit access", nextA == gotA, nextM)
			must(os.Chtimes(path, time.Unix(0, n+789), time.Time{}))
			nextA, nextM = stamp(path)
			fmt.Println("omit modified", nextA, nextM == n+456)
			must(os.Chtimes(path, time.Time{}, time.Time{}))
			lastA, lastM := stamp(path)
			fmt.Println("omit both", lastA == nextA, lastM == nextM)
		}
	}
	now := time.Unix(1700000000, 789)
	failure("absent", filepath.Join(root, "absent"), now, now)
	failure("absent omitted", filepath.Join(root, "absent"), time.Time{}, time.Time{})
	failure("not directory", filepath.Join(file, "child"), now, now)
	failure("nul", file+"\x00tail", now, now)
	failure("nul omitted", file+"\x00tail", time.Time{}, time.Time{})
}
