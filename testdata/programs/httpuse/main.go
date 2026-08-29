package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Kind", "greeting")
		fmt.Fprintf(w, "hello %s", r.URL.Query().Get("name"))
	})
	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "%s:%s:%d", r.Method, string(body), r.ContentLength)
	})
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		io.WriteString(w, "short and stout")
	})
	mux.HandleFunc("/old", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/hello?name=moved", http.StatusFound)
	})
	mux.HandleFunc("/tree/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "tree "+r.URL.Path)
	})
	mux.HandleFunc("/headers", func(w http.ResponseWriter, r *http.Request) {
		keys := make([]string, 0, len(r.Header))
		for k := range r.Header {
			// Go adds Accept-Encoding, because its transport reads gzip. The
			// overlay holds no gzip, thus it adds no such field.
			if k == "User-Agent" || k == "Content-Length" || k == "Accept-Encoding" {
				continue
			}
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(w, "%s=%s\n", k, r.Header.Get(k))
		}
	})

	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		f, ok := w.(http.Flusher)
		io.WriteString(w, "part 1;")
		if ok {
			f.Flush()
		}
		io.WriteString(w, "part 2;")
		if ok {
			f.Flush()
		}
		io.WriteString(w, "end")
	})
	mux.Handle("/strip/", http.StripPrefix("/strip", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, "stripped "+r.URL.Path)
		})))
	mux.HandleFunc("/cookie", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name: "sid", Value: "a b", Path: "/", MaxAge: 60,
			HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode,
		})
		http.SetCookie(w, &http.Cookie{Name: "plain", Value: "v"})
		got, err := r.Cookie("in")
		if err != nil {
			fmt.Fprintf(w, "no cookie: %v", err)
			return
		}
		all := r.Cookies()
		fmt.Fprintf(w, "in=%s count=%d", got.Value, len(all))
	})
	mux.HandleFunc("/big", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Repeat("0123456789", 500)))
	})

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("listen", err)
		return
	}
	server := &http.Server{Handler: mux}
	go server.Serve(l)
	base := "http://" + l.Addr().String()

	show(http.Get(base + "/hello?name=world"))
	show(http.Get(base + "/status"))
	show(http.Get(base + "/missing"))
	show(http.Get(base + "/tree/a/b"))
	show(http.Get(base + "/old"))
	show(http.Post(base+"/echo", "text/plain", strings.NewReader("a body")))
	show(http.PostForm(base+"/echo", url.Values{"k": {"v"}}))
	show(http.Head(base + "/hello?name=head"))

	req, err := http.NewRequest("GET", base+"/headers", nil)
	if err != nil {
		fmt.Println("new request", err)
		return
	}
	req.Header.Set("x-one", "1")
	req.Header.Add("X-Two", "a")
	req.Header.Add("X-Two", "b")
	show(http.DefaultClient.Do(req))

	client := &http.Client{CheckRedirect: func(r *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	show(client.Get(base + "/old"))

	show(http.Get(base + "/stream"))
	show(http.Get(base + "/strip/inner/leaf"))
	if res, err := http.Get(base + "/big"); err == nil {
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		// Go writes a body that is larger than its buffer in the chunked form,
		// and the overlay names the length of each body. Thus the test reads
		// the length of the body only.
		fmt.Println("big", len(body), string(body[:10]))
	} else {
		fmt.Println("big error", err)
	}

	creq, err := http.NewRequest("GET", base+"/cookie", nil)
	if err != nil {
		fmt.Println("cookie request", err)
		return
	}
	creq.AddCookie(&http.Cookie{Name: "in", Value: "one"})
	creq.AddCookie(&http.Cookie{Name: "other", Value: "two"})
	if res, err := http.DefaultClient.Do(creq); err == nil {
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		fmt.Println("cookie body", string(body))
		for _, c := range res.Cookies() {
			fmt.Printf("cookie %s=%q path=%q max=%d http=%v secure=%v same=%d\n",
				c.Name, c.Value, c.Path, c.MaxAge, c.HttpOnly, c.Secure, c.SameSite)
		}
	} else {
		fmt.Println("cookie error", err)
	}

	fmt.Println(http.StatusText(200), http.StatusText(418), http.StatusText(999) == "")
	h := http.Header{}
	h.Add("content-type", "text/html")
	h.Add("Content-Type", "text/plain")
	fmt.Println(h.Get("CONTENT-TYPE"), h.Values("Content-Type"), len(h))
	h.Del("content-type")
	fmt.Println(len(h), h.Get("Content-Type") == "")

	server.Close()
}

// show writes the answer of a request in 1 line.
func show(res *http.Response, err error) {
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("read:", err)
		return
	}
	fmt.Printf("%d %s kind=%q type=%q body=%q\n", res.StatusCode, res.Status,
		res.Header.Get("X-Kind"), res.Header.Get("Content-Type"), string(body))
}
