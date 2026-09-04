//go:build dart

// This file patches the file fe_amd64.go of the package
// crypto/internal/fips140/edwards25519/field.
//
// The real file declares feMul with no body, because assembly holds the code.
// This file holds the body of fe_amd64_noasm.go, thus each caller takes the
// generic path.
package field

func feMul(v, x, y *Element) { feMulGeneric(v, x, y) }
