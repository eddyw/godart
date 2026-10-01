//go:build dart

package os

import (
	"runtime"
	"syscall/dart"
)

type fileResource struct {
	handle int
	dirFD  int
	dir    bool
}

func closeFileResource(resource fileResource) {
	if resource.dir {
		dart.DirectoryClose(resource.dirFD)
	} else {
		dart.FileClose(resource.handle)
	}
}

func fileWithCleanup(f *file) *File {
	f.cleanup = runtime.AddCleanup(f, closeFileResource, fileResource{f.handle, f.dirFD, f.dir})
	return &File{f}
}
