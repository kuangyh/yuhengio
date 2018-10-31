package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func withHTMLSuffix(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := filepath.Base(r.URL.Path)
		if len(name) > 0 && name != "/" && !strings.Contains(name, ".") {
			r.URL.Path += ".html"
		}
		h.ServeHTTP(w, r)
	})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.Handle("/", withHTMLSuffix(http.FileServer(http.Dir("pages"))))

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
