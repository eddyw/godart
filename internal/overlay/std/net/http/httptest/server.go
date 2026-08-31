//go:build dart

package httptest

import (
	"crypto/tls"
	"net"
	"net/http"
	"strings"
)

// A Server holds a listener on a port of the machine and answers each request
// with a handler.
//
// The real Server of Go waits for each request that stands open, and it holds a
// TLS form. Close of the overlay ends the listener and each connection of the
// client that stands open with no request.
type Server struct {
	// URL holds the base address of the server, for example
	// `http://127.0.0.1:52341`.
	URL string
	// Listener holds the listener of the server.
	Listener net.Listener
	// Config holds the server that answers each request.
	Config *http.Server

	client *http.Client
}

// NewServer builds a server and starts it.
func NewServer(handler http.Handler) *Server {
	s := NewUnstartedServer(handler)
	s.Start()
	return s
}

// NewTLSServer builds a server of TLS and starts it.
func NewTLSServer(handler http.Handler) *Server {
	s := NewUnstartedServer(handler)
	s.StartTLS()
	return s
}

// NewUnstartedServer builds a server that holds a listener and answers no
// request. Start answers each request.
func NewUnstartedServer(handler http.Handler) *Server {
	return &Server{
		Listener: newLocalListener(),
		Config:   &http.Server{Handler: handler},
	}
}

// Start answers each request of the listener.
func (s *Server) Start() {
	if s.URL != "" {
		panic("httptest: server already started")
	}
	s.URL = "http://" + s.Listener.Addr().String()
	s.client = &http.Client{}
	go s.Config.Serve(s.Listener)
}

// StartTLS answers each request of the listener over TLS, with the test
// certificate of the package.
//
// Go wraps the listener that the server already holds. The secure socket of
// Dart binds a socket of its own, thus the plain listener closes, and the TLS
// listener takes a fresh port of the loopback address.
func (s *Server) StartTLS() {
	if s.URL != "" {
		panic("httptest: server already started")
	}
	cert, err := tls.X509KeyPair([]byte(testCert), []byte(testKey()))
	if err != nil {
		panic("httptest: " + err.Error())
	}
	s.Listener.Close()
	l, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"http/1.1"},
	})
	if err != nil {
		panic("httptest: failed to listen on a port: " + err.Error())
	}
	s.Listener = l
	s.URL = "https://" + l.Addr().String()
	// The certificate is self signed, and the surface holds no root pool,
	// thus the client of the server drops the check. Go gives its client the
	// certificate as a root.
	s.client = &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}
	go s.Config.Serve(s.Listener)
}

// testCert holds a self signed certificate of the loopback address and of
// localhost, valid to the year 2126. The pair serves tests only.
const testCert = `-----BEGIN CERTIFICATE-----
MIIC/zCCAeegAwIBAgIJAK9noFnHmJSfMA0GCSqGSIb3DQEBCwUAMCUxDzANBgNV
BAoMBmdvZGFydDESMBAGA1UEAwwJbG9jYWxob3N0MCAXDTI2MDgzMTIyMDExNloY
DzIxMjYwODA3MjIwMTE2WjAlMQ8wDQYDVQQKDAZnb2RhcnQxEjAQBgNVBAMMCWxv
Y2FsaG9zdDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAM6gZHPdERlf
6HZ4pCMFHGcLZ6RgAMKgbt6yciWue2qvRBqOaUJLQA0OFp4DCR5nQX0WjwutL2Y9
vytkOibGZpUwhWJB/m/OlErH8pk1TRyqc/QcXkAEqiRoc2iYQ3KQbig/m7ZTMEKb
TZgzvBddVcCA3mLI3HpLA8x8Ipsnxtb5kT6uct0PlwPu8T5jTlyVvMupQjtbuiMC
b26BX7Vlq/nRsfzfp07+Z9rrwc2pCdVdyvl7CqA8aiE5ElX/QpaWqKEG8IM0W41W
ODwBymimb+0KkQOOv+1+N8dsIYQZenKFJakIsHUEXUraCdojvLvKg56+B1hCBKyn
LPzB7/3PrusCAwEAAaMwMC4wLAYDVR0RBCUwI4IJbG9jYWxob3N0hwR/AAABhxAA
AAAAAAAAAAAAAAAAAAABMA0GCSqGSIb3DQEBCwUAA4IBAQATCypJ9ejqEvqUFIAc
fFUcW4ryKAmrHtNpectlRu9pf3EW/fbU9Z2fKzjA8oALL8qu0XX2uwGZ6RWYtt85
nSjOq1cDODcefcvpxTe+qB9y4Pg0xEnAtkjDBufa6TvBHTLvE463tx1+S5SFVGcS
tBILsYVDSdsD1G+9gp5Wuf/TnZN0odhGolhH4q5ufhO6kEtGo+6VNBM4+CFf8497
48VnoleUYEhW/N+/vRcuSKlEidNFjTO1s43Dw57apQDSdpaovljuYC/AMzcnjgrQ
6sUwRDUAUHgngWQREik06DARuX4/dgoBFqKlHFk/PH4K/ioQMWi3mBsTyA0PSNAM
BvWv
-----END CERTIFICATE-----`

// testKey gives the private key of testCert. The text holds the word TESTING,
// thus a scanner of secrets does not read a live key, exactly as the test
// certificate of Go does.
func testKey() string {
	return strings.ReplaceAll(testKeyMarked, "TESTING KEY", "PRIVATE KEY")
}

const testKeyMarked = `-----BEGIN TESTING KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDOoGRz3REZX+h2
eKQjBRxnC2ekYADCoG7esnIlrntqr0QajmlCS0ANDhaeAwkeZ0F9Fo8LrS9mPb8r
ZDomxmaVMIViQf5vzpRKx/KZNU0cqnP0HF5ABKokaHNomENykG4oP5u2UzBCm02Y
M7wXXVXAgN5iyNx6SwPMfCKbJ8bW+ZE+rnLdD5cD7vE+Y05clbzLqUI7W7ojAm9u
gV+1Zav50bH836dO/mfa68HNqQnVXcr5ewqgPGohORJV/0KWlqihBvCDNFuNVjg8
Acpopm/tCpEDjr/tfjfHbCGEGXpyhSWpCLB1BF1K2gnaI7y7yoOevgdYQgSspyz8
we/9z67rAgMBAAECggEAUso7VZ1HI6alCwsNoSaEGdmc/mSlCVak9cAEMor10mpF
XGqhTDnAgTqa31cSCUeQGuG2ERSkQFAPUACJuE2YfsCBQTx2CTB5hHkXAEI2En1N
lorjTL1RuuLLEyanJ7ktGh1ac7RjStzVO13HSIlSOZ6o/GXEyY9TB1TZvNc0OZ/u
9OiJYlyr6j1ToUePBnzpX+x9mNp5x3nlqQrKZQnR9sOi1DMNgR4SrCHsI/X2SWcR
00asTxZrDZF+0YwpybUcqteJBeJ0GSXtF4CSwFLNLeGDcIBhn78N7L6dWPitYjnF
AmV+KuU4cMWKjRuApWc5nmDRG5AZmuZOH7WIsF4b0QKBgQDzPUYgPS48LxI2jgNC
6N17EK9y1VQbxGblD5w0kxBJM0F2SYv2T0B/bKg2vAboyJRJJDLkjLZ4gxBPYmzW
ZLKhKh1nsDqJsUvJsJIuODtbuvui3392o3CswANlobVcz1/VLxHjU6oJznno0yAS
AeeIDmc/L5ln8+8QvxuqTq1agwKBgQDZd2eyKoVUuD8WrMuh1Js3OPbF60xkVYFQ
c8Xyo93uTFV4T9bRoJrmtACugf1vqaLNHyA9fuvZYb+zvhbDN0/rCWi4xAuhpQlT
7PV4wJxk9az/fnbb3WSsZuBOE0DFpLa/i6xVJ6MecP+hQAuxWL0eAFTqQdueFiAm
3fxLOPfNeQKBgD4clG6hENBfeLgjlWOdkQje3t5k6SBaC6hMTygBsbPvebTZ3Pqd
jCu9tEqdwunoZi0zKCfCYsntNdn/mSk5938+ANo8XwZQFrk+9GUHCOlImv8hMSw7
WgdS1Ms7+tLjRWZrqZayA4aTif/RAax3fgb5HpY88mBgItjU5ByeEJg/AoGAOn85
IJ75gKGTLulKWXzUR9WF5BV4ViD+CQMrgyp7P2e/Mf7f/UGjiV1wNYD4CHpsNZsi
1Af5L19w7su6WcbeM44zYkgyDMhVKwM5wMKnDy1a3NtK47e5JfsCaV4adU/qbLoI
oFtqMH0a9mhA/PKu5IwYS29twSeYfHwIKr/nQJkCgYEAyMxUkCT2C/qDfqeCLv4/
irn4zBlfRSM6My2WEK+eTAM0Z6LvkwwgvPLxLuzZJGfBc9CXzEYl3/8SswXxFYyV
GCgH6kOX2blj7DqiYOqmYsU47ll2Wn19QGp01CZ4++uJhGQ5ONHMdrBv9aBDKU8Q
BChx2JXwiXwYE525nl9qsTw=
-----END TESTING KEY-----`

// Close ends the listener and each connection of the client that stands open
// with no request.
func (s *Server) Close() {
	s.Config.Close()
	if s.client != nil {
		s.client.CloseIdleConnections()
	}
}

// Client gives a client that reaches the server.
func (s *Server) Client() *http.Client {
	if s.client == nil {
		s.client = &http.Client{}
	}
	return s.client
}

// newLocalListener binds a port of the loopback address.
func newLocalListener() net.Listener {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic("httptest: failed to listen on a port: " + err.Error())
	}
	return l
}
