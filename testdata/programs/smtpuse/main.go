// The program smtpuse drives the client of net/smtp against a server of its
// own. The server binds port 0 and the program reads the real address, thus
// the output holds no port of the machine.
package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

func main() {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("listen:", err)
		return
	}
	addr := ln.Addr().String()
	done := make(chan []string, 4)

	go serve(ln, done)
	session(addr)
	fmt.Println("== server 1 ==")
	report(<-done)

	go serve(ln, done)
	auth(addr)
	fmt.Println("== server 2 ==")
	report(<-done)

	go serve(ln, done)
	cram(addr)
	fmt.Println("== server 3 ==")
	report(<-done)

	go serve(ln, done)
	send(addr)
	fmt.Println("== server 4 ==")
	report(<-done)

	ln.Close()
	values()
}

// report writes each line that the server read.
func report(lines []string) {
	for _, l := range lines {
		fmt.Println(" ", l)
	}
}

// serve holds 1 SMTP session of a server. It writes each command that it read
// into the channel at the end.
func serve(ln net.Listener, done chan []string) {
	c, err := ln.Accept()
	if err != nil {
		done <- []string{"accept: " + err.Error()}
		return
	}
	defer c.Close()
	var log []string
	r := bufio.NewReader(c)
	w := bufio.NewWriter(c)
	write := func(s string) {
		w.WriteString(s + "\r\n")
		w.Flush()
	}
	write("220 mail.example.com ESMTP ready")
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimRight(line, "\r\n")
		log = append(log, line)
		verb, rest, _ := strings.Cut(line, " ")
		switch strings.ToUpper(verb) {
		case "EHLO":
			write("250-mail.example.com Hello " + rest)
			write("250-AUTH PLAIN CRAM-MD5")
			write("250 8BITMIME")
		case "HELO":
			write("250 mail.example.com")
		case "AUTH":
			mech, initial, _ := strings.Cut(rest, " ")
			switch strings.ToUpper(mech) {
			case "PLAIN":
				raw, err := base64.StdEncoding.DecodeString(initial)
				if err != nil {
					write("501 bad base64")
					continue
				}
				log = append(log, "  plain="+strings.ReplaceAll(string(raw), "\x00", "|"))
				write("235 accepted")
			case "CRAM-MD5":
				// The challenge stays fixed, thus the answer of the client
				// stays equal on any machine.
				write("334 " + base64.StdEncoding.EncodeToString([]byte("<1234@example.com>")))
				answer, err := r.ReadString('\n')
				if err != nil {
					return
				}
				raw, err := base64.StdEncoding.DecodeString(strings.TrimRight(answer, "\r\n"))
				if err != nil {
					write("501 bad base64")
					continue
				}
				log = append(log, "  cram="+string(raw))
				write("235 accepted")
			default:
				write("504 unknown mechanism")
			}
		case "MAIL", "RCPT", "RSET", "NOOP", "VRFY":
			write("250 ok")
		case "DATA":
			write("354 send it")
			for {
				body, err := r.ReadString('\n')
				if err != nil {
					return
				}
				body = strings.TrimRight(body, "\r\n")
				if body == "." {
					break
				}
				log = append(log, "  body="+body)
			}
			write("250 queued")
		case "QUIT":
			write("221 bye")
			done <- log
			return
		default:
			write("500 unknown command")
		}
	}
	done <- log
}

// session drives the plain command set of the client.
func session(addr string) {
	c, err := smtp.Dial(addr)
	if err != nil {
		fmt.Println("dial:", err)
		return
	}
	fmt.Println("== session ==")
	fmt.Println("hello", c.Hello("client.example.org"))
	ok, param := c.Extension("AUTH")
	fmt.Printf("ext AUTH %v %q\n", ok, param)
	ok, param = c.Extension("8BITMIME")
	fmt.Printf("ext 8BITMIME %v %q\n", ok, param)
	ok, param = c.Extension("SIZE")
	fmt.Printf("ext SIZE %v %q\n", ok, param)
	fmt.Println("noop", c.Noop())
	fmt.Println("verify", c.Verify("bob@example.net"))
	fmt.Println("mail", c.Mail("alice@example.com"))
	fmt.Println("rcpt", c.Rcpt("bob@example.net"))
	w, err := c.Data()
	fmt.Println("data", err)
	if err == nil {
		fmt.Fprintf(w, "Subject: a test\r\n\r\nthe body\r\n.a dot line\r\n")
		fmt.Println("close", w.Close())
	}
	fmt.Println("reset", c.Reset())
	fmt.Println("quit", c.Quit())
}

// auth drives PlainAuth.
func auth(addr string) {
	c, err := smtp.Dial(addr)
	if err != nil {
		fmt.Println("dial:", err)
		return
	}
	fmt.Println("== auth ==")
	a := smtp.PlainAuth("", "alice", "secret", "127.0.0.1")
	fmt.Println("auth", c.Auth(a))
	fmt.Println("quit", c.Quit())
}

// cram drives CRAMMD5Auth.
func cram(addr string) {
	c, err := smtp.Dial(addr)
	if err != nil {
		fmt.Println("dial:", err)
		return
	}
	fmt.Println("== cram ==")
	fmt.Println("auth", c.Auth(smtp.CRAMMD5Auth("alice", "secret")))
	fmt.Println("quit", c.Quit())
}

// send drives SendMail.
func send(addr string) {
	fmt.Println("== sendmail ==")
	msg := []byte("Subject: hello\r\n\r\nthe whole message\r\n")
	err := smtp.SendMail(addr, smtp.PlainAuth("", "alice", "secret", "127.0.0.1"),
		"alice@example.com", []string{"bob@example.net"}, msg)
	fmt.Println("sendmail", err)
}

// values reads each name of the package that needs no server.
func values() {
	fmt.Println("== values ==")
	info := &smtp.ServerInfo{Name: "127.0.0.1", TLS: false, Auth: []string{"PLAIN"}}
	proto, resp, err := smtp.PlainAuth("id", "alice", "secret", "127.0.0.1").Start(info)
	fmt.Printf("plain start %q %q %v\n", proto, strings.ReplaceAll(string(resp), "\x00", "|"), err)
	remote := &smtp.ServerInfo{Name: "mail.example.com", TLS: false}
	_, _, err = smtp.PlainAuth("", "alice", "secret", "mail.example.com").Start(remote)
	fmt.Println("plain remote", err)
	cr := smtp.CRAMMD5Auth("alice", "secret")
	proto, resp, err = cr.Start(info)
	fmt.Printf("cram start %q %q %v\n", proto, string(resp), err)
	answer, err := cr.Next([]byte("<1234@example.com>"), true)
	fmt.Printf("cram next %q %v\n", string(answer), err)
	answer, err = cr.Next(nil, false)
	fmt.Printf("cram done %q %v\n", string(answer), err)
	// A line with a newline never reaches the server.
	err = smtp.SendMail("127.0.0.1:1", nil, "a\nb@example.com", nil, nil)
	fmt.Println("bad line", err)
}
