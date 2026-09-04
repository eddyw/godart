// Command aesuse reads the AES block cipher and the modes of crypto/cipher.
// Each answer below comes from a published test vector.
package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func unhex(s string) []byte {
	b, err := hex.DecodeString(s)
	must(err)
	return b
}

func main() {
	// FIPS 197, appendix C. 1 block of 16 bytes, with a key of 128, of 192,
	// and of 256 bits.
	plain := unhex("00112233445566778899aabbccddeeff")
	keys := []string{
		"000102030405060708090a0b0c0d0e0f",
		"000102030405060708090a0b0c0d0e0f1011121314151617",
		"000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
	}
	for _, k := range keys {
		block, err := aes.NewCipher(unhex(k))
		must(err)
		fmt.Println(block.BlockSize())
		out := make([]byte, aes.BlockSize)
		block.Encrypt(out, plain)
		fmt.Println(hex.EncodeToString(out))
		back := make([]byte, aes.BlockSize)
		block.Decrypt(back, out)
		fmt.Println(hex.EncodeToString(back))
	}

	// NIST SP 800-38A, F.2.1 and F.2.2. CBC with the key of 128 bits.
	key := unhex("2b7e151628aed2a6abf7158809cf4f3c")
	iv := unhex("000102030405060708090a0b0c0d0e0f")
	text := unhex("6bc1bee22e409f96e93d7e117393172a" +
		"ae2d8a571e03ac9c9eb76fac45af8e51" +
		"30c81c46a35ce411e5fbc1191a0a52ef" +
		"f69f2445df4f9b17ad2b417be66c3710")
	block, err := aes.NewCipher(key)
	must(err)
	ct := make([]byte, len(text))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ct, text)
	fmt.Println(hex.EncodeToString(ct))
	pt := make([]byte, len(ct))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(pt, ct)
	fmt.Println(hex.EncodeToString(pt))

	// NIST SP 800-38A, F.5.1. CTR with the key of 128 bits.
	ctr := unhex("f0f1f2f3f4f5f6f7f8f9fafbfcfdfeff")
	out := make([]byte, len(text))
	cipher.NewCTR(block, ctr).XORKeyStream(out, text)
	fmt.Println(hex.EncodeToString(out))

	// NIST SP 800-38A, F.3.13. CFB with a segment of 128 bits.
	out2 := make([]byte, len(text))
	cipher.NewCFBEncrypter(block, iv).XORKeyStream(out2, text)
	fmt.Println(hex.EncodeToString(out2))

	// NIST SP 800-38A, F.4.1. OFB with the key of 128 bits.
	out3 := make([]byte, len(text))
	cipher.NewOFB(block, iv).XORKeyStream(out3, text)
	fmt.Println(hex.EncodeToString(out3))

	// The test case 4 of the GCM specification of McGrew and Viega.
	gkey := unhex("feffe9928665731c6d6a8f9467308308")
	gnonce := unhex("cafebabefacedbaddecaf888")
	gplain := unhex("d9313225f88406e5a55909c5aff5269a" +
		"86a7a9531534f7da2e4c303d8a318a72" +
		"1c3c0c95956809532fcf0e2449a6b525" +
		"b16aedf5aa0de657ba637b39")
	gaad := unhex("feedfacedeadbeeffeedfacedeadbeefabaddad2")
	gblock, err := aes.NewCipher(gkey)
	must(err)
	gcm, err := cipher.NewGCM(gblock)
	must(err)
	fmt.Println(gcm.NonceSize(), gcm.Overhead())
	sealed := gcm.Seal(nil, gnonce, gplain, gaad)
	fmt.Println(hex.EncodeToString(sealed))
	opened, err := gcm.Open(nil, gnonce, sealed, gaad)
	must(err)
	fmt.Println(hex.EncodeToString(opened))
	sealed[0] ^= 1
	_, err = gcm.Open(nil, gnonce, sealed, gaad)
	fmt.Println(err)

	_, err = aes.NewCipher(make([]byte, 7))
	fmt.Println(err)
}
