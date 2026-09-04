package main

import (
	"encoding/base32"
	"fmt"
	"strings"
)

func main() {
	text := "the quick brown fox"
	encoded := base32.StdEncoding.EncodeToString([]byte(text))
	fmt.Println("encoded:", encoded)
	decoded, err := base32.StdEncoding.DecodeString(encoded)
	fmt.Println("decoded:", string(decoded), err == nil)

	hex := base32.HexEncoding.EncodeToString([]byte("ab"))
	fmt.Println("hex:", hex)
	raw := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte("ab"))
	fmt.Println("raw:", raw)
	fmt.Println("len:", base32.StdEncoding.EncodedLen(9), base32.StdEncoding.DecodedLen(16))

	if _, err := base32.StdEncoding.DecodeString("!!!!!!!!"); err != nil {
		fmt.Println("broken:", err)
	}

	var out strings.Builder
	w := base32.NewEncoder(base32.StdEncoding, &out)
	w.Write([]byte("stream of bytes"))
	w.Close()
	fmt.Println("stream:", out.String())

	r := base32.NewDecoder(base32.StdEncoding, strings.NewReader(out.String()))
	buf := make([]byte, 32)
	n, _ := r.Read(buf)
	fmt.Println("back:", string(buf[:n]))
}
