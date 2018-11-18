package main

import (
	"fmt"
	"html/template"
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

var goRedirectHTML = template.Must(template.New("goRedirect").Parse(`
<html>
<head>
	<title>Package yuheng.io/r/{{.Path}}</title>
	<meta name="go-import" content="yuheng.io/r/ git https://github.com/kuangyh"></meta>
</head>
<body>
<h1>Package yuheng.io/r/{{.Path}}</h1>
<ul>
	<li><a href="https://godoc.org/yuheng.io/r/{{.Path}}">Document</a></li>
</ul>
</body>
</html>
	`))

func redirectGoPackage(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/r/") {
		http.Error(w, "package not found", 404)
	}
	data := struct {
		Path string
	}{
		Path: r.URL.Path[3:],
	}
	if err := goRedirectHTML.Execute(w, data); err != nil {
		http.Error(w, "wtf", 500)
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.Handle("/r/", http.HandlerFunc(redirectGoPackage))
	http.Handle("/", withHTMLSuffix(http.FileServer(http.Dir("pages"))))

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
