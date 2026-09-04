// Command cryptoiface reads the constants and the interfaces of the package
// crypto. The known answer is the name and the size of each hash.
package main

import (
	"crypto"
	"fmt"
)

func main() {
	hashes := []crypto.Hash{
		crypto.MD4, crypto.MD5, crypto.SHA1, crypto.SHA224, crypto.SHA256,
		crypto.SHA384, crypto.SHA512, crypto.MD5SHA1, crypto.RIPEMD160,
		crypto.SHA3_224, crypto.SHA3_256, crypto.SHA3_384, crypto.SHA3_512,
		crypto.SHA512_224, crypto.SHA512_256, crypto.BLAKE2s_256,
		crypto.BLAKE2b_256, crypto.BLAKE2b_384, crypto.BLAKE2b_512,
	}
	for _, h := range hashes {
		fmt.Printf("%d %s %d\n", uint(h), h.String(), h.Size())
	}
	fmt.Println(crypto.Hash(99).String())
	var opts crypto.SignerOpts = crypto.SHA256
	fmt.Println(opts.HashFunc() == crypto.SHA256)
	fmt.Println(crypto.SHA256.Available())
}
