//go:build dart

// This file patches the file indicator.go of the package
// crypto/internal/fips140.
//
// The real file holds the service indicator in a value of the goroutine that
// the runtime of Go owns, and it reaches that value through a go:linkname
// directive. The scheduler of gart holds no such value, thus this file holds
// the indicator in a variable of the package.
//
// The surface reaches no name of this package. crypto/internal/fips140test
// alone reads the indicator, and that package stands outside the surface, thus
// no program observes the difference.
package fips140

var indicator uint8

func getIndicator() uint8 { return indicator }

func setIndicator(v uint8) { indicator = v }

const (
	indicatorUnset uint8 = iota
	indicatorFalse
	indicatorTrue
)

// ResetServiceIndicator clears the service indicator.
func ResetServiceIndicator() {
	setIndicator(indicatorUnset)
}

// ServiceIndicator reports whether each service that ran since the last call of
// ResetServiceIndicator is approved.
func ServiceIndicator() bool {
	return getIndicator() == indicatorTrue
}

// RecordApproved records the use of an approved service. It does not replace a
// call of RecordNonApproved of the same span.
func RecordApproved() {
	if getIndicator() == indicatorUnset {
		setIndicator(indicatorTrue)
	}
}

// RecordNonApproved records the use of a service that is not approved.
func RecordNonApproved() {
	setIndicator(indicatorFalse)
}
