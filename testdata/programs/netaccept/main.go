package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
)

func main() {
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	certificate := server.TLS.Certificates[0]
	server.Close()
	for _, protocol := range []string{"tcp", "tls"} {
		var listener net.Listener
		var err error
		if protocol == "tls" {
			listener, err = tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{certificate}})
		} else {
			listener, err = net.Listen("tcp", "127.0.0.1:0")
		}
		if err != nil {
			panic(err)
		}
		exercise(protocol, listener)
	}
}

func exercise(protocol string, listener net.Listener) {
	const clients = 8
	results := make(chan error, clients)
	for range clients {
		go func() {
			conn, err := listener.Accept()
			if err == nil {
				_, err = conn.Write([]byte{1})
				conn.Close()
			}
			results <- err
		}()
	}
	for range clients {
		var conn net.Conn
		var err error
		if protocol == "tls" {
			conn, err = tls.Dial("tcp", listener.Addr().String(), &tls.Config{InsecureSkipVerify: true})
		} else {
			conn, err = net.Dial("tcp", listener.Addr().String())
		}
		if err != nil {
			panic(err)
		}
		var response [1]byte
		_, err = io.ReadFull(conn, response[:])
		conn.Close()
		if err != nil {
			panic(err)
		}
		if response[0] != 1 {
			panic("invalid response")
		}
	}
	for range clients {
		if err := <-results; err != nil {
			panic(err)
		}
	}
	ready := make(chan bool)
	for range clients {
		go func() {
			ready <- true
			conn, err := listener.Accept()
			if conn != nil {
				conn.Close()
			}
			results <- err
		}()
	}
	for range clients {
		<-ready
	}
	listener.Close()
	for range clients {
		if err := <-results; err == nil {
			panic("closed listener accepted a connection")
		}
	}
	fmt.Println(protocol, clients, "accepted", clients, "closed")
}
