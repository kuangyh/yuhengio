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
	goProjects  []string
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("go-get") == "1" {
		for _, p := range s.goProjects {
			if !strings.HasPrefix(r.URL.Path[1:], p) {
				continue
			}
			w.Write([]byte(fmt.Sprintf(`<meta name="go-import" content="yuheng.io/%s git https://github.com/kuangyh/%s">`, p, p)))
			return
		}
		http.Error(w, "package not found", 404)
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
		goProjects: []string{
			"yuhengio",
			"swiffy",
		},
	}
	http.Handle("/", s)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
