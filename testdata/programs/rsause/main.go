// Command rsause reads the signature and the encryption of crypto/rsa. The key
// below holds 2048 bits, and it stays the same at each run. Each answer comes
// from a fixed input.
package main

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
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

func num(s string) *big.Int {
	n, ok := new(big.Int).SetString(s, 16)
	if !ok {
		panic("bad number")
	}
	return n
}

const (
	nHex = "cf2f45608d51a436ab8851312f989122eefc86ae4c61317678784dbf4cb5ec17" +
		"23a66dffb3612ebeeb445119d0fad6b232fdfe5953f97bd5d3f48d0b59860a9d" +
		"ead21c89989e08d42f322cdadc8383a0373fa5ba98d0706a00953f8c3a1b493a" +
		"2c29188c590c5f514abc5f36db8840dd52ca11bac29c6d4734d7736f92045332" +
		"5c070c1b72f66d835fae1ca3d3db7474123e6da5accbc28b53d49f9e6aca4a21" +
		"0ccff309346e5276369fb4c11842625637cc39860adaed082706148db2e27d34" +
		"23ab164c16fc5dca8926de15ce1f0c866a5df27f3e0e30a5cfb447689959332b" +
		"9fe33788570bd73b9a14dab87c9349d8dc9251c05c97a4ec79c6eb0b7fe44419"
	dHex = "eeee04d9797552d0a40b6b6c33078483aa3819844871387ef779f059fce28fdc" +
		"df856a0ac8435035675191cb1945d848095100fce3402e4a449fc7489f7ab023" +
		"b84f8c0ee25788b6ffa14d8e835f921c379ac99fe1e5ff6894e3898af2e27a15" +
		"ac227892378a1d2267f31f9ae946be0729c2a9a4f21493b1852edce1289df43c" +
		"bbbbbcba98d9b3c07a7913dbf178e1a92f9d7becd888c3fe0a3f40503e042b55" +
		"95f91e7cc385219b5a63ff5ff3c9792009e8549aafc280545d5e22870d125d53" +
		"e39f9f1b832b84658d01282f95498f0e5dbbe58b7f560a237e207dd9f7b17465" +
		"08152b44bf886cb69bde59f6ccaf6d022cbba3fc8d840450b74bb339632be55"
	pHex = "dd393540730a72cb3f373115f965bd6325ebf526d0365367086090216ebf3568" +
		"43896e630bb84ce31c6e7ba0af39d31c634b52f2f2e23fcde6b891e95621c396" +
		"1bf03bff1a37d336d03fcd74290537cb83381a645f73776dafb56221ca67fc8e" +
		"60e1066e7a2a66f0414d690c83cc43ad9cf5fb9ec1c47248229856a2959a082f"
	qHex = "efc117c2854cb612db06b4a32b8e3ca1d634bb8de06e3ea6710071c82d4f5130" +
		"4cdda2a73c7320a70aa895029c3be92cfe5e38868aad0e5712a03668711dcc13" +
		"a16a61b700e61de03dbaf7f859041a20957ebfd18b84990c98a8248a889825d3" +
		"f0e5955d5a7a7fa2848d4f1889d19d0f9cb0755fc79699f544d2698ad1d91e37"
)

func main() {
	priv := &rsa.PrivateKey{
		PublicKey: rsa.PublicKey{N: num(nHex), E: 65537},
		D:         num(dHex),
		Primes:    []*big.Int{num(pHex), num(qHex)},
	}
	priv.Precompute()
	must(priv.Validate())
	fmt.Println(priv.Size(), priv.N.BitLen(), priv.E, len(priv.Primes))

	msg := []byte("the message that the program signs")
	sum := sha256.Sum256(msg)
	other := sha256.Sum256([]byte("another message"))

	// PKCS 1 version 1.5 holds a fixed padding, thus the signature never
	// changes.
	sig, err := rsa.SignPKCS1v15(nil, priv, crypto.SHA256, sum[:])
	must(err)
	fmt.Println(hex.EncodeToString(sig))
	fmt.Println(rsa.VerifyPKCS1v15(&priv.PublicKey, crypto.SHA256, sum[:], sig))
	fmt.Println(rsa.VerifyPKCS1v15(&priv.PublicKey, crypto.SHA256, other[:], sig))

	// PSS holds a random salt, thus only the check gives a fixed answer.
	pss, err := rsa.SignPSS(rand.Reader, priv, crypto.SHA256, sum[:], nil)
	must(err)
	fmt.Println(rsa.VerifyPSS(&priv.PublicKey, crypto.SHA256, sum[:], pss, nil))
	fmt.Println(rsa.VerifyPSS(&priv.PublicKey, crypto.SHA256, other[:], pss, nil))

	// OAEP, and the round trip of the message.
	ct, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, &priv.PublicKey, msg, []byte("label"))
	must(err)
	pt, err := rsa.DecryptOAEP(sha256.New(), nil, priv, ct, []byte("label"))
	must(err)
	fmt.Println(bytes.Equal(pt, msg))
	_, err = rsa.DecryptOAEP(sha256.New(), nil, priv, ct, []byte("other"))
	fmt.Println(err)

	// PKCS 1 version 1.5 encryption, and the round trip of the message.
	ct2, err := rsa.EncryptPKCS1v15(rand.Reader, &priv.PublicKey, msg)
	must(err)
	pt2, err := rsa.DecryptPKCS1v15(nil, priv, ct2)
	must(err)
	fmt.Println(bytes.Equal(pt2, msg))

	// The Signer and the Decrypter of the package crypto.
	var signer crypto.Signer = priv
	sig2, err := signer.Sign(nil, sum[:], crypto.SHA256)
	must(err)
	fmt.Println(bytes.Equal(sig, sig2))
	var dec crypto.Decrypter = priv
	pt3, err := dec.Decrypt(nil, ct2, nil)
	must(err)
	fmt.Println(bytes.Equal(pt3, msg))

	// A key of 2048 bits that the program builds signs and verifies.
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	must(err)
	must(k.Validate())
	fmt.Println(k.Size(), k.E, len(k.Primes))
	sig3, err := rsa.SignPKCS1v15(nil, k, crypto.SHA256, sum[:])
	must(err)
	fmt.Println(rsa.VerifyPKCS1v15(&k.PublicKey, crypto.SHA256, sum[:], sig3))
	fmt.Println(k.Equal(k), k.PublicKey.Equal(&k.PublicKey))

	// A key that holds too few bits gives an error.
	_, err = rsa.GenerateKey(rand.Reader, 128)
	fmt.Println(err)
}
