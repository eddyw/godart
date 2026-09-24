//go:build dart && !((linux || darwin) && (amd64 || arm64))

package os

import "io"

func (f *File) readDirNames(n int) ([]string, []FileMode, error) {
	all, err := ReadNames(f.name)
	if err != nil {
		return nil, nil, err
	}
	if f.dirPos > len(all) {
		f.dirPos = len(all)
	}
	rest := all[f.dirPos:]
	if n > 0 && len(rest) == 0 {
		return []string{}, nil, io.EOF
	}
	if n > 0 && n < len(rest) {
		rest = rest[:n]
	}
	f.dirPos += len(rest)
	types := make([]FileMode, len(rest))
	for i := range types {
		types[i] = ^FileMode(0)
	}
	return rest, types, nil
}

func (f *File) directoryInfo(name string) (FileInfo, error) { return Lstat(f.name + "/" + name) }

func openDirectory(name string) (*File, error) { return Open(name) }
