package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
)

func get(url string) (int, string, string) {
	res, err := http.Get(url)
	if err != nil {
		return 0, "", err.Error()
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, res.Header.Get("Content-Type"), string(b)
}

func main() {
	dir, err := os.MkdirTemp("", "serve")
	if err != nil {
		fmt.Println("temp:", err)
		return
	}
	defer os.RemoveAll(dir)
	os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello world"), 0o644)
	os.WriteFile(filepath.Join(dir, "page.html"), []byte("<h1>hi</h1>"), 0o644)
	os.MkdirAll(filepath.Join(dir, "sub"), 0o755)
	os.WriteFile(filepath.Join(dir, "sub", "deep.txt"), []byte("deep"), 0o644)

	srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
	defer srv.Close()

	fmt.Println(get(srv.URL + "/hello.txt"))
	fmt.Println(get(srv.URL + "/page.html"))
	fmt.Println(get(srv.URL + "/sub/deep.txt"))
	code, _, _ := get(srv.URL + "/nothere.txt")
	fmt.Println("missing", code)
	code, _, _ = get(srv.URL + "/../hello.txt")
	fmt.Println("escape", code)

	// FileServerFS reads an fs.FS.
	srv2 := httptest.NewServer(http.FileServerFS(os.DirFS(dir)))
	defer srv2.Close()
	fmt.Println(get(srv2.URL + "/hello.txt"))

	// StripPrefix and ServeFile.
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(dir))))
	mux.HandleFunc("/one", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(dir, "hello.txt"))
	})
	srv3 := httptest.NewServer(mux)
	defer srv3.Close()
	fmt.Println(get(srv3.URL + "/static/hello.txt"))
	fmt.Println(get(srv3.URL + "/one"))

	fmt.Println(http.DetectContentType([]byte("<html><body>x")))
	fmt.Println(http.DetectContentType([]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}))
	fmt.Println(http.DetectContentType([]byte{0, 1, 2, 3}))
}
