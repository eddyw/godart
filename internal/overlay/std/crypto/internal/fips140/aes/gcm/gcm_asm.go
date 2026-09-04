//go:build dart

// This file patches the file gcm_asm.go of the package
// crypto/internal/fips140/aes/gcm.
//
// The real file reaches an instruction of the machine. This file holds the body
// of gcm_noasm.go, thus each caller takes the generic path.
package gcm

func checkGenericIsExpected() {}

type gcmPlatformData struct{}

func initGCM(g *GCM) {}

func seal(out []byte, g *GCM, nonce, plaintext, data []byte) {
	sealGeneric(out, g, nonce, plaintext, data)
}

func open(out []byte, g *GCM, nonce, ciphertext, data []byte) error {
	return openGeneric(out, g, nonce, ciphertext, data)
}
