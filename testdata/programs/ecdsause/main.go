// Command ecdsause reads the signature of crypto/ecdsa. The key and the answer
// below come from the example of the appendix of FIPS 186-4 for P-256, and from
// the test vectors of RFC 6979.
package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func mustInt(s string) *big.Int {
	n, ok := new(big.Int).SetString(s, 16)
	if !ok {
		panic("bad number")
	}
	return n
}

func main() {
	// The key of the test vector of RFC 6979, appendix A.2.5, over P-256.
	d := mustInt("C9AFA9D845BA75166B5C215767B1D6934E50C3DB36E89B127B8A622B120F6721")
	priv := new(ecdsa.PrivateKey)
	priv.Curve = elliptic.P256()
	priv.D = d
	priv.X, priv.Y = priv.Curve.ScalarBaseMult(d.Bytes())
	fmt.Println(priv.X.Text(16))
	fmt.Println(priv.Y.Text(16))

	// The public key of the vector.
	wantX := mustInt("60FED4BA255A9D31C961EB74C6356D68C049B8923B61FA6CE669622E60F29FB6")
	wantY := mustInt("7903FE1008B8BC99A41AE9E95628BC64F2F1B20C2D7E9F5177A3C294D4462299")
	fmt.Println(priv.X.Cmp(wantX) == 0, priv.Y.Cmp(wantY) == 0)

	// The signature of the message "sample" of the vector, and the check of it.
	msg := []byte("sample")
	sum := sha256.Sum256(msg)
	r, s, err := ecdsa.Sign(rand.Reader, priv, sum[:])
	must(err)
	fmt.Println(ecdsa.Verify(&priv.PublicKey, sum[:], r, s))

	// The signature of the vector itself passes the check.
	vr := mustInt("EFD48B2AACB6A8FD1140DD9CD45E81D69D2C877B56AAF991C34D0EA84EAF3716")
	vs := mustInt("F7CB1C942D657C41D436C7A1B6E29F65F3E900DBB9AFF4064DC4AB2F843ACDA8")
	fmt.Println(ecdsa.Verify(&priv.PublicKey, sum[:], vr, vs))

	// A signature of another digest fails the check.
	other := sha256.Sum256([]byte("test"))
	fmt.Println(ecdsa.Verify(&priv.PublicKey, other[:], vr, vs))

	// The ASN.1 form of a signature, and the check of it.
	der, err := ecdsa.SignASN1(rand.Reader, priv, sum[:])
	must(err)
	fmt.Println(ecdsa.VerifyASN1(&priv.PublicKey, sum[:], der))
	fmt.Println(ecdsa.VerifyASN1(&priv.PublicKey, other[:], der))

	// The Signer interface of the package crypto.
	var signer crypto.Signer = priv
	sig, err := signer.Sign(rand.Reader, sum[:], crypto.SHA256)
	must(err)
	fmt.Println(ecdsa.VerifyASN1(&priv.PublicKey, sum[:], sig))

	// A new key of each curve signs and verifies.
	for _, c := range []elliptic.Curve{elliptic.P224(), elliptic.P256(),
		elliptic.P384(), elliptic.P521()} {
		k, err := ecdsa.GenerateKey(c, rand.Reader)
		must(err)
		fmt.Println(k.Curve.Params().Name, k.PublicKey.Curve == c)
		sg, err := ecdsa.SignASN1(rand.Reader, k, sum[:])
		must(err)
		fmt.Println(ecdsa.VerifyASN1(&k.PublicKey, sum[:], sg))
		fmt.Println(k.Equal(k), k.PublicKey.Equal(&k.PublicKey))
	}

	// The byte form of a key and the read of it again.
	b, err := priv.Bytes()
	must(err)
	fmt.Println(hex.EncodeToString(b))
	p2, err := ecdsa.ParseRawPrivateKey(elliptic.P256(), b)
	must(err)
	fmt.Println(p2.Equal(priv))
	pb, err := priv.PublicKey.Bytes()
	must(err)
	fmt.Println(hex.EncodeToString(pb))
	q2, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), pb)
	must(err)
	fmt.Println(q2.Equal(&priv.PublicKey))
}
