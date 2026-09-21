// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package testing

import (
	"fmt"
	"os"
	"strings"
	"unicode"
)

func (t *T) newTempDir(who string) string {
	dir, err := t.makeTempDir()
	if err != nil {
		t.Fatalf("%s: %v", who, err)
	}
	return dir
}

func (t *T) makeTempDir() (string, error) {
	t.tempDirMu.Lock()
	defer t.tempDirMu.Unlock()
	missing := t.tempDir == ""
	if !missing {
		_, err := os.Stat(t.tempDir)
		missing = os.IsNotExist(err)
		if err != nil && !missing {
			return "", err
		}
	}
	if missing {
		pattern := t.Name()
		pattern = pattern[:min(len(pattern), 64)]
		const allowed = "!#$%&()+,-.=@^_{}~ "
		pattern = strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsNumber(r) || strings.ContainsRune(allowed, r) {
				return r
			}
			return -1
		}, pattern)
		dir, err := os.MkdirTemp(os.Getenv("GOTMPDIR"), pattern)
		if err != nil {
			return "", err
		}
		t.tempDir = dir
		t.Cleanup(func() {
			if err := os.RemoveAll(dir); err != nil {
				t.Errorf("TempDir RemoveAll cleanup: %v", err)
			}
		})
	}
	t.tempDirSeq++
	dir := fmt.Sprintf("%s%c%03d", t.tempDir, os.PathSeparator, t.tempDirSeq)
	if err := os.Mkdir(dir, 0o777); err != nil {
		return "", err
	}
	return dir, nil
}
