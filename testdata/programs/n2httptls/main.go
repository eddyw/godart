// Command n2httptls proves the TLS listener of the server, the Dialer of
// crypto/tls, the cross origin protection, and the trailer of an answer. The
// sixteenth session added each 1.
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	dir, err := os.MkdirTemp("", "n2httptls")
	if err != nil {
		fmt.Println("temp:", err)
		return
	}
	defer os.RemoveAll(dir)
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")
	if err := os.WriteFile(certFile, []byte(testCert), 0o600); err != nil {
		fmt.Println("write cert:", err)
		return
	}
	key := strings.ReplaceAll(testKeyMarked, "TESTING KEY", "PRIVATE KEY")
	if err := os.WriteFile(keyFile, []byte(key), 0o600); err != nil {
		fmt.Println("write key:", err)
		return
	}

	keyPair(certFile, keyFile)
	serveTLS(certFile, keyFile)
	protocolRejections()
	crossOrigin()
	trailers()
	optionalNames()
}

// keyPair proves LoadX509KeyPair.
func keyPair(certFile, keyFile string) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		fmt.Println("load pair:", err)
		return
	}
	fmt.Println("pair:", len(cert.Certificate), cert.Leaf.Subject.Organization[0])
	_, err = tls.LoadX509KeyPair(certFile+".missing", keyFile)
	fmt.Println("missing file rejected:", err != nil)
}

// freePort takes a port of the system and gives it back at once. The listener
// of the test then binds a port that no other program holds.
func freePort() (string, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	addr := l.Addr().String()
	l.Close()
	return addr, nil
}

// serveTLS proves Server.ListenAndServeTLS, Server.TLSConfig,
// Server.Protocols, and the Dialer of crypto/tls.
func serveTLS(certFile, keyFile string) {
	addr, err := freePort()
	if err != nil {
		fmt.Println("port:", err)
		return
	}
	var only http.Protocols
	only.SetHTTP1(true)
	mux := http.NewServeMux()
	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		_, isPusher := w.(http.Pusher)
		fmt.Fprintf(w, "secure %s tls=%t push=%t", r.URL.Path, r.TLS != nil, isPusher)
	})
	server := &http.Server{
		Addr:      addr,
		Handler:   mux,
		Protocols: &only,
		TLSConfig: &tls.Config{},
		// A dial that ends before the handshake makes the server of Go log a
		// failure at a time of its own. The log thus reaches no output.
		ErrorLog: log.New(io.Discard, "", 0),
	}
	go func() {
		err := server.ListenAndServeTLS(certFile, keyFile)
		if err != nil && err != http.ErrServerClosed {
			fmt.Println("serve:", err)
		}
	}()
	defer server.Close()

	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}
	var res *http.Response
	for i := 0; i < 100; i++ {
		res, err = client.Get("https://" + addr + "/hello")
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		fmt.Println("get:", err)
		return
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println("get:", res.StatusCode, string(body))

	// The Dialer of crypto/tls names the protocol of ALPN.
	dialer := &tls.Dialer{
		NetDialer: &net.Dialer{Timeout: 5 * time.Second},
		Config:    &tls.Config{InsecureSkipVerify: true, NextProtos: []string{"http/1.1"}},
	}
	conn, err := dialer.DialContext(context.Background(), "tcp", addr)
	if err != nil {
		fmt.Println("dial:", err)
		return
	}
	secure := conn.(*tls.Conn)
	state := secure.ConnectionState()
	fmt.Println("alpn:", state.NegotiatedProtocol, state.NegotiatedProtocolIsMutual,
		state.HandshakeComplete)
	fmt.Println("handshake again:", secure.HandshakeContext(context.Background()))
	conn.Close()

	// A context that already ended gives no connection.
	ended, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = dialer.DialContext(ended, "tcp", addr)
	fmt.Println("ended context rejected:", err != nil)
}

// protocolRejections proves the check of the set of protocols on the
// transport.
//
// The server holds the same check. Go carries HTTP/2 over TCP, and it then
// serves the listener with no end. Thus this witness leaves that path out and
// docs/parity.md names it.
func protocolRejections() {
	var plain http.Protocols
	plain.SetUnencryptedHTTP2(true)
	transport := &http.Transport{Protocols: &plain}
	_, err := transport.RoundTrip(mustRequest("https://127.0.0.1:1/x"))
	fmt.Println("transport unencrypted h2 rejected:", err != nil)
}

// mustRequest builds a request of a URL, and it stops the program for a URL
// that holds no form.
func mustRequest(url string) *http.Request {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	return req
}

// crossOrigin proves each answer of CrossOriginProtection.Check.
func crossOrigin() {
	protection := http.NewCrossOriginProtection()
	if err := protection.AddTrustedOrigin("https://good.example"); err != nil {
		fmt.Println("trusted:", err)
		return
	}
	fmt.Println("bad origin:", protection.AddTrustedOrigin("good.example"))

	cases := []struct {
		name   string
		method string
		site   string
		origin string
	}{
		{"safe method", "GET", "cross-site", ""},
		{"same origin", "POST", "same-origin", ""},
		{"none", "POST", "none", ""},
		{"cross site", "POST", "cross-site", ""},
		{"trusted origin", "POST", "cross-site", "https://good.example"},
		{"no site header", "POST", "", "http://127.0.0.1"},
		{"other origin", "POST", "", "https://other.example"},
	}
	for _, c := range cases {
		req := httptest.NewRequest(c.method, "http://127.0.0.1/form", nil)
		if c.site != "" {
			req.Header.Set("Sec-Fetch-Site", c.site)
		}
		if c.origin != "" {
			req.Header.Set("Origin", c.origin)
		}
		fmt.Printf("csrf %-16s %v\n", c.name, protection.Check(req))
	}
}

// trailers proves TrailerPrefix and the field Trailer of the head.
func trailers() {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("listen:", err)
		return
	}
	defer l.Close()
	server := &http.Server{Handler: http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Trailer", "Sum")
			io.WriteString(w, "body")
			w.Header().Set("Sum", "42")
			w.Header().Set(http.TrailerPrefix+"Late", "yes")
		})}
	go server.Serve(l)
	defer server.Close()

	conn, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		fmt.Println("dial:", err)
		return
	}
	fmt.Fprint(conn, "GET /t HTTP/1.1\r\nHost: x\r\nConnection: close\r\n\r\n")
	raw, err := io.ReadAll(conn)
	conn.Close()
	if err != nil {
		fmt.Println("read:", err)
		return
	}
	text := string(raw)
	fmt.Println("chunked:", strings.Contains(text, "Transfer-Encoding: chunked"))
	fmt.Println("declares:", strings.Contains(text, "Trailer: Sum"))
	fmt.Println("prefix out of head:", !strings.Contains(text, "Trailer:Late"))
	fmt.Println("sum after body:", strings.Contains(text, "Sum: 42"))
	fmt.Println("late after body:", strings.Contains(text, "Late: yes"))
	fmt.Println("sum after zero chunk:",
		strings.Index(text, "Sum: 42") > strings.Index(text, "\r\n0\r\n"))
}

// optionalNames proves that the 3 optional names of the package hold a type.
func optionalNames() {
	var pusher http.Pusher
	var notifier http.CloseNotifier
	options := http.PushOptions{Method: "GET", Header: http.Header{"A": {"1"}}}
	fmt.Println("optional:", pusher == nil, notifier == nil,
		options.Method, options.Header.Get("A"))
	fmt.Println("prefix:", http.TrailerPrefix)
}

// testCert holds the certificate of net/http/internal/testcert. It names
// 127.0.0.1, ::1, example.com, and *.example.com. The pair serves tests only.
const testCert = `-----BEGIN CERTIFICATE-----
MIIDSDCCAjCgAwIBAgIQEP/md970HysdBTpuzDOf0DANBgkqhkiG9w0BAQsFADAS
MRAwDgYDVQQKEwdBY21lIENvMCAXDTcwMDEwMTAwMDAwMFoYDzIwODQwMTI5MTYw
MDAwWjASMRAwDgYDVQQKEwdBY21lIENvMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A
MIIBCgKCAQEAxcl69ROJdxjN+MJZnbFrYxyQooADCsJ6VDkuMyNQIix/Hk15Nk/u
FyBX1Me++aEpGmY3RIY4fUvELqT/srvAHsTXwVVSttMcY8pcAFmXSqo3x4MuUTG/
jCX3Vftj0r3EM5M8ImY1rzA/jqTTLJg00rD+DmuDABcqQvoXw/RV8w1yTRi5BPoH
DFD/AWTt/YgMvk1l2Yq/xI8VbMUIpjBoGXxWsSevQ5i2s1mk9/yZzu0Ysp1tTlzD
qOPa4ysFjBitdXiwfxjxtv5nXqOCP5rheKO0sWLk0fetMp1OV5JSJMAJw6c2ZMkl
U2WMqAEpRjdE/vHfIuNg+yGaRRqI07NZRQIDAQABo4GXMIGUMA4GA1UdDwEB/wQE
AwICpDATBgNVHSUEDDAKBggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MB0GA1Ud
DgQWBBQR5QIzmacmw78ZI1C4MXw7Q0wJ1jA9BgNVHREENjA0ggtleGFtcGxlLmNv
bYINKi5leGFtcGxlLmNvbYcEfwAAAYcQAAAAAAAAAAAAAAAAAAAAATANBgkqhkiG
9w0BAQsFAAOCAQEACrRNgiioUDzxQftd0fwOa6iRRcPampZRDtuaF68yNHoNWbOu
LUwc05eOWxRq3iABGSk2xg+FXM3DDeW4HhAhCFptq7jbVZ+4Jj6HeJG9mYRatAxR
Y/dEpa0D0EHhDxxVg6UzKOXB355n0IetGE/aWvyTV9SiDs6QsaC57Q9qq1/mitx5
2GFBoapol9L5FxCc77bztzK8CpLujkBi25Vk6GAFbl27opLfpyxkM+rX/T6MXCPO
6/YBacNZ7ff1/57Etg4i5mNA6ubCpuc4Gi9oYqCNNohftr2lkJr7REdDR6OW0lsL
rF7r4gUnKeC7mYIH1zypY7laskopiLFAfe96Kg==
-----END CERTIFICATE-----`

// testKeyMarked holds the private key of testCert. The text holds the word
// TESTING, thus a scanner of secrets does not read a live key.
const testKeyMarked = `-----BEGIN TESTING KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDFyXr1E4l3GM34
wlmdsWtjHJCigAMKwnpUOS4zI1AiLH8eTXk2T+4XIFfUx775oSkaZjdEhjh9S8Qu
pP+yu8AexNfBVVK20xxjylwAWZdKqjfHgy5RMb+MJfdV+2PSvcQzkzwiZjWvMD+O
pNMsmDTSsP4Oa4MAFypC+hfD9FXzDXJNGLkE+gcMUP8BZO39iAy+TWXZir/EjxVs
xQimMGgZfFaxJ69DmLazWaT3/JnO7RiynW1OXMOo49rjKwWMGK11eLB/GPG2/mde
o4I/muF4o7SxYuTR960ynU5XklIkwAnDpzZkySVTZYyoASlGN0T+8d8i42D7IZpF
GojTs1lFAgMBAAECggEAIYthUi1lFBDd5gG4Rzlu+BlBIn5JhcqkCqLEBiJIFfOr
/4yuMRrvS3bNzqWt6xJ9MSAC4ZlN/VobRLnxL/QNymoiGYUKCT3Ww8nvPpPzR9OE
sE68TUL9tJw/zZJcRMKwgvrGqSLimfq53MxxkE+kLdOc0v9C8YH8Re26mB5ZcWYa
7YFyZQpKsQYnsmu/05cMbpOQrQWhtmIqRoyn8mG/par2s3NzjtpSE9NINyz26uFc
k/3ovFJQIHkUmTS7KHD3BgY5vuCqP98HramYnOysJ0WoYgvSDNCWw3037s5CCwJT
gCKuM+Ow6liFrj83RrdKBpm5QUGjfNpYP31o+QNP4QKBgQDSrUQ2XdgtAnibAV7u
7kbxOxro0EhIKso0Y/6LbDQgcXgxLqltkmeqZgG8nC3Z793lhlSasz2snhzzooV5
5fTy1y8ikXqjhG0nNkInFyOhsI0auE28CFoDowaQd+5cmCatpN4Grqo5PNRXxm1w
HktfPEgoP11NNCFHvvN5fEKbbQKBgQDwVlOaV20IvW3IPq7cXZyiyabouFF9eTRo
VJka1Uv+JtyvL2P0NKkjYHOdN8gRblWqxQtJoTNk020rVA4UP1heiXALy50gvj/p
hMcybPTLYSPOhAGx838KIcvGR5oskP1aUCmFbFQzGELxhJ9diVVjxUtbG2DuwPKd
tD9TLxT2OQKBgQCcdlHSjp+dzdgERmBa0ludjGfPv9/uuNizUBAbO6D690psPFtY
JQMYaemgSd1DngEOFVWADt4e9M5Lose+YCoqr+UxpxmNlyv5kzJOFcFAs/4XeglB
PHKdgNW/NVKxMc6H54l9LPr+x05sYdGlEtqnP/3W5jhEvhJ5Vjc8YiyVgQKBgQCl
zwjyrGo+42GACy7cPYE5FeIfIDqoVByB9guC5bD98JXEDu/opQQjsgFRcBCJZhOY
M0UsURiB8ROaFu13rpQq9KrmmF0ZH+g8FSzQbzcbsTLg4VXCDXmR5esOKowFPypr
Sm667BfTAGP++D5ya7MLmCv6+RKQ5XD8uEQQAaV2kQKBgAD8qeJuWIXZT0VKkQrn
nIhgtzGERF/6sZdQGW2LxTbUDWG74AfFkkEbeBfwEkCZXY/xmnYqYABhvlSex8jU
supU6Eea21esIxIub2zv/Np0ojUb6rlqTPS4Ox1E27D787EJ3VOXpriSD10vyNnZ
jel6uj2FOP9g54s+GzlSVg/T
-----END TESTING KEY-----`
