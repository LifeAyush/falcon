// Command backend is a minimal upstream for benchmarks: it answers every
// request with a small fixed body so the proxy is the only variable.
package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	addr := flag.String("addr", ":3101", "listen address")
	flag.Parse()

	body := []byte("ok\n")
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write(body)
	})
	log.Printf("bench backend on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, h))
}
