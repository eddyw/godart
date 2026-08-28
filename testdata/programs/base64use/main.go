package main

import (
	"encoding/base64"
	"fmt"
)

func main() {
	text := "the quick brown fox"
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	fmt.Println("encoded:", encoded)
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	fmt.Println("decoded:", string(decoded), err == nil)

	url := base64.URLEncoding.EncodeToString([]byte{0xfb, 0xff, 0x00})
	fmt.Println("url:", url)
	raw := base64.RawStdEncoding.EncodeToString([]byte("ab"))
	fmt.Println("raw:", raw)

	if _, err := base64.StdEncoding.DecodeString("not base64!!"); err == nil {
		fmt.Println("the broken text passed")
	} else {
		fmt.Println("broken:", err != nil)
	}
}
