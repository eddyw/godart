// httpmiddle shows the middleware family of net/http: the handler of a time
// limit, the reader of a limit of bytes, the handler of a redirect, the reader
// of the semicolon of a query, the empty body, and the errors of the package.
package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

func main() {
	timeout()
	maxBytes()
	redirect()
	semicolons()
	noBody()
	errorValues()
}

// timeout reads TimeoutHandler and ErrHandlerTimeout.
func timeout() {
	late := make(chan bool, 1)
	slow := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		// The time ran out, thus each write gives ErrHandlerTimeout.
		_, err := io.WriteString(w, "late")
		late <- errors.Is(err, http.ErrHandlerTimeout)
	})
	h := http.TimeoutHandler(slow, 50*time.Millisecond, "took too long")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	res := w.Result()
	body, _ := io.ReadAll(res.Body)
	fmt.Printf("timeout %d %q\n", res.StatusCode, string(body))
	fmt.Println("timeout write:", <-late)

	fast := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Tag", "fast")
		w.WriteHeader(http.StatusAccepted)
		io.WriteString(w, "in time")
	})
	w2 := httptest.NewRecorder()
	http.TimeoutHandler(fast, time.Second, "").ServeHTTP(w2,
		httptest.NewRequest("GET", "/", nil))
	res2 := w2.Result()
	body2, _ := io.ReadAll(res2.Body)
	fmt.Printf("timeout fast %d %q tag=%s\n", res2.StatusCode, string(body2),
		res2.Header.Get("X-Tag"))
}

// maxBytes reads MaxBytesReader, MaxBytesHandler, and MaxBytesError.
func maxBytes() {
	w := httptest.NewRecorder()
	body := io.NopCloser(strings.NewReader("0123456789"))
	limited := http.MaxBytesReader(w, body, 4)
	read, err := io.ReadAll(limited)
	fmt.Printf("maxBytes %q err=%v\n", string(read), err)
	var mbe *http.MaxBytesError
	if errors.As(err, &mbe) {
		fmt.Println("maxBytes limit:", mbe.Limit, "text:", mbe.Error())
	}
	fmt.Println("maxBytes close:", limited.Close())

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		read, err := io.ReadAll(r.Body)
		fmt.Printf("  handler %q err=%v\n", string(read), err)
	})
	h := http.MaxBytesHandler(inner, 3)
	req := httptest.NewRequest("POST", "/", strings.NewReader("abcdef"))
	h.ServeHTTP(httptest.NewRecorder(), req)

	// A negative limit reads as 0.
	zero := http.MaxBytesReader(nil, io.NopCloser(strings.NewReader("x")), -1)
	_, err = io.ReadAll(zero)
	fmt.Println("maxBytes zero:", err)
}

// redirect reads RedirectHandler.
func redirect() {
	h := http.RedirectHandler("/moved", http.StatusMovedPermanently)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/old", nil))
	res := w.Result()
	body, _ := io.ReadAll(res.Body)
	fmt.Printf("redirect %d %s %q\n", res.StatusCode, res.Header.Get("Location"),
		string(body))
}

// semicolons reads AllowQuerySemicolons.
func semicolons() {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("  query a=%q b=%q raw=%q uri=%q\n", r.URL.Query().Get("a"),
			r.URL.Query().Get("b"), r.URL.RawQuery, r.RequestURI)
	})
	h := http.AllowQuerySemicolons(inner)
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/p?a=1;b=2", nil))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/p?a=1&b=2", nil))
}

// noBody reads NoBody.
func noBody() {
	n, err := http.NoBody.Read(make([]byte, 4))
	fmt.Println("noBody read", n, err == io.EOF, http.NoBody.Close())
	req, _ := http.NewRequest("POST", "http://example.com/", http.NoBody)
	fmt.Println("noBody request", req.ContentLength, req.Body == http.NoBody)
}

// errorValues reads the error values of the package.
func errorValues() {
	list := []struct {
		name string
		err  error
	}{
		{"ErrAbortHandler", http.ErrAbortHandler},
		{"ErrBodyNotAllowed", http.ErrBodyNotAllowed},
		{"ErrBodyReadAfterClose", http.ErrBodyReadAfterClose},
		{"ErrContentLength", http.ErrContentLength},
		{"ErrHandlerTimeout", http.ErrHandlerTimeout},
		{"ErrHeaderTooLong", http.ErrHeaderTooLong},
		{"ErrHijacked", http.ErrHijacked},
		{"ErrLineTooLong", http.ErrLineTooLong},
		{"ErrMissingBoundary", http.ErrMissingBoundary},
		{"ErrMissingContentLength", http.ErrMissingContentLength},
		{"ErrMissingFile", http.ErrMissingFile},
		{"ErrNoCookie", http.ErrNoCookie},
		{"ErrNoLocation", http.ErrNoLocation},
		{"ErrNotMultipart", http.ErrNotMultipart},
		{"ErrNotSupported", http.ErrNotSupported},
		{"ErrSchemeMismatch", http.ErrSchemeMismatch},
		{"ErrServerClosed", http.ErrServerClosed},
		{"ErrShortBody", http.ErrShortBody},
		{"ErrSkipAltProtocol", http.ErrSkipAltProtocol},
		{"ErrUnexpectedTrailer", http.ErrUnexpectedTrailer},
		{"ErrUseLastResponse", http.ErrUseLastResponse},
		{"ErrWriteAfterFlush", http.ErrWriteAfterFlush},
	}
	for _, e := range list {
		fmt.Printf("%s = %q\n", e.name, e.err.Error())
	}
	fmt.Println("ErrNotSupported is ErrUnsupported:",
		errors.Is(http.ErrNotSupported, errors.ErrUnsupported))
	pe := &http.ProtocolError{ErrorString: "made here"}
	fmt.Println("ProtocolError:", pe.Error(), pe.Is(errors.ErrUnsupported))
}
