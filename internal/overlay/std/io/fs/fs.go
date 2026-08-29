//go:build dart

// Package fs holds the types of a file system that os and path/filepath share.
//
// The overlay holds no FS interface and no read of an embedded file system. It
// holds the types that a walk of a directory tree and a stat of a path need.
package fs

import (
	"errors"
	"time"
)

// A FileMode holds the type of a file and its permission bits.
type FileMode uint32

// The bits of a FileMode. The type bits stand at the top, and the permission
// bits stand at the bottom.
const (
	ModeDir FileMode = 1 << (32 - 1 - iota)
	ModeAppend
	ModeExclusive
	ModeTemporary
	ModeSymlink
	ModeDevice
	ModeNamedPipe
	ModeSocket
	ModeSetuid
	ModeSetgid
	ModeCharDevice
	ModeSticky
	ModeIrregular

	// ModeType holds each bit that names a type of file.
	ModeType = ModeDir | ModeSymlink | ModeNamedPipe | ModeSocket | ModeDevice |
		ModeCharDevice | ModeIrregular

	// ModePerm holds the 9 permission bits.
	ModePerm FileMode = 0777
)

// IsDir reports whether the mode names a directory.
func (m FileMode) IsDir() bool { return m&ModeDir != 0 }

// IsRegular reports whether the mode names a plain file.
func (m FileMode) IsRegular() bool { return m&ModeType == 0 }

// Perm gives the 9 permission bits of the mode.
func (m FileMode) Perm() FileMode { return m & ModePerm }

// Type gives the bits that name the type of the file.
func (m FileMode) Type() FileMode { return m & ModeType }

// modeLetters pairs each type bit and each attribute bit with its letter, in
// the order that String writes them.
const modeLetters = "dalTLDpSugct?"

var modeBits = []FileMode{
	ModeDir, ModeAppend, ModeExclusive, ModeTemporary, ModeSymlink,
	ModeDevice, ModeNamedPipe, ModeSocket, ModeSetuid, ModeSetgid,
	ModeCharDevice, ModeSticky, ModeIrregular,
}

// String gives the text of the mode, as in `drwxr-xr-x`.
func (m FileMode) String() string {
	out := ""
	for i, bit := range modeBits {
		if m&bit != 0 {
			out += string(rune(modeLetters[i]))
		}
	}
	if out == "" {
		out = "-"
	}
	const rwx = "rwxrwxrwx"
	for i := 0; i < 9; i++ {
		if m&(1<<uint(8-i)) != 0 {
			out += string(rune(rwx[i]))
		} else {
			out += "-"
		}
	}
	return out
}

// A FileInfo holds the metadata of a file.
type FileInfo interface {
	Name() string
	Size() int64
	Mode() FileMode
	ModTime() time.Time
	IsDir() bool
	Sys() any
}

// A DirEntry holds 1 member of a directory.
type DirEntry interface {
	Name() string
	IsDir() bool
	Type() FileMode
	Info() (FileInfo, error)
}

// A PathError names the operation, the path, and the cause.
type PathError struct {
	Op   string
	Path string
	Err  error
}

func (e *PathError) Error() string { return e.Op + " " + e.Path + ": " + e.Err.Error() }

func (e *PathError) Unwrap() error { return e.Err }

// The errors that a file operation gives.
var (
	ErrInvalid    = errors.New("invalid argument")
	ErrPermission = errors.New("permission denied")
	ErrExist      = errors.New("file already exists")
	ErrNotExist   = errors.New("file does not exist")
	ErrClosed     = errors.New("file already closed")
)

// SkipDir tells a walk to leave the directory that holds the current entry.
var SkipDir = errors.New("skip this directory")

// SkipAll tells a walk to stop with no error.
var SkipAll = errors.New("skip everything and stop the walk")

// A WalkDirFunc runs for each file and each directory of a walk.
type WalkDirFunc func(path string, d DirEntry, err error) error

// dirInfo holds a FileInfo as a DirEntry.
type dirInfo struct {
	info FileInfo
}

func (d dirInfo) Name() string            { return d.info.Name() }
func (d dirInfo) IsDir() bool             { return d.info.IsDir() }
func (d dirInfo) Type() FileMode          { return d.info.Mode().Type() }
func (d dirInfo) Info() (FileInfo, error) { return d.info, nil }

// FileInfoToDirEntry gives a DirEntry that reads the FileInfo.
func FileInfoToDirEntry(info FileInfo) DirEntry {
	if info == nil {
		return nil
	}
	return dirInfo{info: info}
}

// ValidPath reports whether the name fits the rules of a path of an FS. Such a
// path holds no leading slash, no trailing slash, and no element `.` or `..`.
func ValidPath(name string) bool {
	if name == "." {
		return true
	}
	if name == "" {
		return false
	}
	for {
		i := 0
		for i < len(name) && name[i] != '/' {
			i++
		}
		elem := name[:i]
		if elem == "" || elem == "." || elem == ".." {
			return false
		}
		if i == len(name) {
			return true
		}
		name = name[i+1:]
	}
}
