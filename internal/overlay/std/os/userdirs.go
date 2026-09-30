// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package os

import (
	"errors"
	"path"
	"runtime"
)

func UserCacheDir() (string, error) {
	var dir string

	switch runtime.GOOS {
	case "darwin", "ios":
		dir = Getenv("HOME")
		if dir == "" {
			return "", errors.New("$HOME is not defined")
		}
		dir += "/Library/Caches"

	default:
		dir = Getenv("XDG_CACHE_HOME")
		if dir == "" {
			dir = Getenv("HOME")
			if dir == "" {
				return "", errors.New("neither $XDG_CACHE_HOME nor $HOME are defined")
			}
			dir += "/.cache"
		} else if !path.IsAbs(dir) {
			return "", errors.New("path in $XDG_CACHE_HOME is relative")
		}
	}

	return dir, nil
}

func UserConfigDir() (string, error) {
	var dir string

	switch runtime.GOOS {
	case "darwin", "ios":
		dir = Getenv("HOME")
		if dir == "" {
			return "", errors.New("$HOME is not defined")
		}
		dir += "/Library/Application Support"

	default:
		dir = Getenv("XDG_CONFIG_HOME")
		if dir == "" {
			dir = Getenv("HOME")
			if dir == "" {
				return "", errors.New("neither $XDG_CONFIG_HOME nor $HOME are defined")
			}
			dir += "/.config"
		} else if !path.IsAbs(dir) {
			return "", errors.New("path in $XDG_CONFIG_HOME is relative")
		}
	}

	return dir, nil
}
