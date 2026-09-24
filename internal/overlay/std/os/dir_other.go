//go:build dart && !((linux || darwin) && (amd64 || arm64))

package os

import "io"

func (f *File) readDirNames(n int) ([]string, error) {
	all, err := ReadNames(f.name)
	if err != nil {
		return nil, err
	}
	if f.dirPos > len(all) {
		f.dirPos = len(all)
	}
	rest := all[f.dirPos:]
	if n <= 0 {
		f.dirPos = len(all)
		return rest, nil
	}
	if len(rest) == 0 {
		return nil, io.EOF
	}
	if n < len(rest) {
		rest = rest[:n]
	}
	f.dirPos += len(rest)
	return rest, nil
}
