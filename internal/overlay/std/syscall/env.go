// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

// Unix environment variables.

package syscall

import "syscall/dart"

var (
	// env maps from an environment variable to its first occurrence in envs.
	env map[string]int

	// The runtime supplies envs. Elements are expected to
	// be of the form "key=value". An empty string means deleted
	// (or a duplicate to be ignored).
	envs []string = dart.Environ()
)

// Environment operations cannot suspend within the single Dart isolate.
func copyenv() {
	if env != nil {
		return
	}
	env = make(map[string]int)
	for i, s := range envs {
		for j := 0; j < len(s); j++ {
			if s[j] == '=' {
				key := s[:j]
				if _, ok := env[key]; !ok {
					env[key] = i // first mention of key
				} else {
					// Clear duplicate keys. This permits Unsetenv to
					// safely delete only the first item without
					// worrying about unshadowing a later one,
					// which might be a security problem.
					envs[i] = ""
				}
				break
			}
		}
	}
}

func Unsetenv(key string) error {
	copyenv()

	if i, ok := env[key]; ok {
		envs[i] = ""
		delete(env, key)
	}
	if key == "GODEBUG" {
		dart.SetRuntimeGODEBUG("")
	}
	return nil
}

func Getenv(key string) (value string, found bool) {
	copyenv()
	if len(key) == 0 {
		return "", false
	}

	i, ok := env[key]
	if !ok {
		return "", false
	}
	s := envs[i]
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return s[i+1:], true
		}
	}
	return "", false
}

func Setenv(key, value string) error {
	copyenv()
	if len(key) == 0 {
		return EINVAL
	}
	for i := 0; i < len(key); i++ {
		if key[i] == '=' || key[i] == 0 {
			return EINVAL
		}
	}
	for i := 0; i < len(value); i++ {
		if value[i] == 0 {
			return EINVAL
		}
	}

	i, ok := env[key]
	kv := key + "=" + value
	if ok {
		envs[i] = kv
	} else {
		i = len(envs)
		envs = append(envs, kv)
	}
	env[key] = i
	if key == "GODEBUG" {
		dart.SetRuntimeGODEBUG(value)
	}
	return nil
}

func Clearenv() {
	copyenv()

	env = make(map[string]int)
	envs = []string{}
	dart.SetRuntimeGODEBUG("")
}

func Environ() []string {
	copyenv()
	a := make([]string, 0, len(envs))
	for _, env := range envs {
		if env != "" {
			a = append(a, env)
		}
	}
	return a
}
