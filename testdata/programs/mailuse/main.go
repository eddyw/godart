// The program mailuse reads each exported name of net/mail. It parses an
// address, a list of addresses, a date, and a whole message. The output holds
// no time of the machine and no path of the machine.
package main

import (
	"fmt"
	"io"
	"mime"
	"net/mail"
	"sort"
	"strings"
)

func main() {
	single()
	list()
	dates()
	parser()
	message()
	format()
}

// single parses 1 address and reports the name and the address.
func single() {
	fmt.Println("== single ==")
	for _, in := range []string{
		"Barry Gibbs <bg@example.com>",
		"<bg@example.com>",
		"bg@example.com",
		"\"Gibbs, Barry\" <bg@example.com>",
		"=?utf-8?q?Bj=C3=B6rn?= <bj@example.com>",
		"Barry Gibbs",
		"",
	} {
		a, err := mail.ParseAddress(in)
		if err != nil {
			fmt.Printf("%q -> error %v\n", in, err)
			continue
		}
		fmt.Printf("%q -> name=%q address=%q\n", in, a.Name, a.Address)
	}
}

// list parses a list of addresses.
func list() {
	fmt.Println("== list ==")
	for _, in := range []string{
		"Alice <a@example.com>, Bob <b@example.net>",
		"group: a@example.com, b@example.com;",
		"a@example.com,",
	} {
		as, err := mail.ParseAddressList(in)
		if err != nil {
			fmt.Printf("%q -> error %v\n", in, err)
			continue
		}
		var parts []string
		for _, a := range as {
			parts = append(parts, a.String())
		}
		fmt.Printf("%q -> %s\n", in, strings.Join(parts, " | "))
	}
}

// dates parses a date of a header.
func dates() {
	fmt.Println("== dates ==")
	for _, in := range []string{
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"Thu, 20 Nov 1997 14:22:01 -0800 (PST)",
		"21 Nov 97 09:55:06 GMT",
		"not a date",
	} {
		t, err := mail.ParseDate(in)
		if err != nil {
			fmt.Printf("%q -> error %v\n", in, err)
			continue
		}
		fmt.Printf("%q -> %s unix=%d\n", in, t.Format("2006-01-02T15:04:05Z07:00"), t.Unix())
	}
}

// parser reads an AddressParser with a word decoder of its own.
func parser() {
	fmt.Println("== parser ==")
	p := &mail.AddressParser{WordDecoder: &mime.WordDecoder{
		CharsetReader: func(charset string, input io.Reader) (io.Reader, error) {
			if charset != "x-caps" {
				return nil, fmt.Errorf("unknown charset %q", charset)
			}
			b, err := io.ReadAll(input)
			if err != nil {
				return nil, err
			}
			return strings.NewReader(strings.ToUpper(string(b))), nil
		},
	}}
	a, err := p.Parse("=?x-caps?q?hello?= <h@example.com>")
	fmt.Println(a, err)
	as, err := p.ParseList("=?x-caps?q?one?= <1@example.com>, =?x-caps?q?two?= <2@example.com>")
	for _, x := range as {
		fmt.Println(" ", x)
	}
	fmt.Println("err", err)
}

// message reads a whole message and each name of Header.
func message() {
	fmt.Println("== message ==")
	const src = "From: Alice <a@example.com>\r\n" +
		"To: Bob <b@example.net>, Carol <c@example.org>\r\n" +
		"Subject: =?utf-8?q?a_test?=\r\n" +
		"Date: Mon, 02 Jan 2006 15:04:05 -0700\r\n" +
		"X-Fold: one\r\n" +
		" two\r\n" +
		"\r\n" +
		"the body\r\n"
	msg, err := mail.ReadMessage(strings.NewReader(src))
	if err != nil {
		fmt.Println("read:", err)
		return
	}
	keys := make([]string, 0, len(msg.Header))
	for k := range msg.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("%s=%q\n", k, msg.Header.Get(k))
	}
	d, err := msg.Header.Date()
	fmt.Println("date", d.Format("2006-01-02T15:04:05Z07:00"), err)
	to, err := msg.Header.AddressList("To")
	for _, a := range to {
		fmt.Println("to", a)
	}
	fmt.Println("to err", err)
	_, err = msg.Header.AddressList("Cc")
	fmt.Println("cc err", err, err == mail.ErrHeaderNotPresent)
	body, err := io.ReadAll(msg.Body)
	fmt.Printf("body=%q err=%v\n", string(body), err)
}

// format writes an Address again as text.
func format() {
	fmt.Println("== format ==")
	for _, a := range []mail.Address{
		{Name: "Barry Gibbs", Address: "bg@example.com"},
		{Name: "", Address: "bg@example.com"},
		{Name: "Gibbs, Barry", Address: "bg@example.com"},
		{Name: "Björn", Address: "bj@example.com"},
		{Name: "tab\there", Address: "t@example.com"},
	} {
		fmt.Println(a.String())
	}
}
