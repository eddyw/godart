//go:build dart

// This file patches the file sha3_amd64.go of the package
// crypto/internal/fips140/sha3.
//
// The real file reaches an instruction of the machine. This file holds the body
// of sha3_noasm.go, thus each caller takes the generic path.
package sha3

func keccakF1600(a *[200]byte) {
	keccakF1600Generic(a)
}

func (d *Digest) write(p []byte) (n int, err error) {
	return d.writeGeneric(p)
}

func (d *Digest) read(out []byte) (n int, err error) {
	return d.readGeneric(out)
}

func (d *Digest) sum(b []byte) []byte {
	return d.sumGeneric(b)
}
