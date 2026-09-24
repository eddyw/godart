// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package poll

type errNetClosing struct{}

func (e errNetClosing) Error() string   { return "use of closed network connection" }
func (e errNetClosing) Timeout() bool   { return false }
func (e errNetClosing) Temporary() bool { return false }

var ErrNetClosing = errNetClosing{}
