//go:build dart

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file patches the file arith_decl.go of the package math/big.
//
// The real file declares 6 functions with no body, because an assembly file
// of each architecture holds the code. This file supplies Go bodies and
// runtime vector arithmetic, thus the package needs no assembly.
//
// The real file also carries a go:linkname directive for each name, because a
// third party module reaches the name that way. The Dart output holds no such
// link, thus the directives stay out.
package big

import "syscall/dart"

func addVV(z, x, y []Word) (c Word) { return addVV_g(z, x, y) }

func subVV(z, x, y []Word) (c Word) { return subVV_g(z, x, y) }

func lshVU(z, x []Word, s uint) (c Word) { return lshVU_g(z, x, s) }

func rshVU(z, x []Word, s uint) (c Word) { return rshVU_g(z, x, s) }

func mulAddVWW(z, x []Word, m, a Word) (c Word) {
	if len(x) != len(z) {
		panic("mulAddVWW len")
	}
	if len(z) == 0 {
		return a
	}
	return Word(dart.BigMulAddVWW((*uint)(&z[0]), (*uint)(&x[0]), len(z), uint(m), uint(a)))
}

func addMulVVWW(z, x, y []Word, m, a Word) (c Word) {
	if len(x) != len(z) || len(y) != len(z) {
		panic("addMulVVWW len")
	}
	if len(z) == 0 {
		return a
	}
	return Word(dart.BigAddMulVVWW((*uint)(&z[0]), (*uint)(&x[0]), (*uint)(&y[0]), len(z), uint(m), uint(a)))
}

// shlVU and addMulVVW each hold a body in the real file, and a third party
// module reaches the 2 through a link. The 2 stay, because a caller inside the
// package reads them.

func shlVU(z, x []Word, s uint) (c Word) {
	if s == 0 {
		copy(z, x)
		return 0
	}
	return lshVU(z, x, s)
}

func addMulVVW(z, x []Word, y Word) (c Word) {
	return addMulVVWW(z, z, x, y, 0)
}
