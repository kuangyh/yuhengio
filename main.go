package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type server struct {
	pageHandler http.Handler
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("go-get") == "1" {
		if r.URL.Path == "/" {
			http.Error(w, "no package", 404)
			return
		}
		// assume first section of the path is repository name
		repo := strings.Split(r.URL.Path[1:], "/")[0]
		w.Write([]byte(fmt.Sprintf(`<meta name="go-import" content="yuheng.io/%s git https://github.com/kuangyh/%s">`, repo, repo)))
		return
	}
	name := filepath.Base(r.URL.Path)
	if len(name) > 0 && name != "/" && !strings.Contains(name, ".") {
		r.URL.Path += ".html"
	}
	s.pageHandler.ServeHTTP(w, r)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	s := &server{
		pageHandler: http.FileServer(http.Dir("pages")),
	}
	http.Handle("/", s)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
