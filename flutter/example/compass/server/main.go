// Command compassserver holds the server of the Flutter sample compass_app.
// It answers the same 9 routes with the same JSON, and the Dart client of the
// sample reads it with no change.
//
// gart compiles it to Dart, thus the server runs on the Dart VM as well as on
// Go. The 2 runs must give the same answer.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	assets := flag.String("assets", "../assets", "the directory of the 2 JSON files")
	port := flag.Int("port", 8080, "the port of the server")
	flag.Parse()
	if v := os.Getenv("PORT"); v != "" {
		fmt.Sscan(v, port)
	}
	s, err := newServer(*assets)
	if err != nil {
		log.Fatal(err)
	}
	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	fmt.Printf("Server listening on port %d\n", *port)
	log.Fatal(http.ListenAndServe(addr, s.handler()))
}
