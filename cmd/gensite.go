package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	blackfriday "github.com/russross/blackfriday/v2"
)

var spaces = regexp.MustCompile("\\s+")

type varMap map[string]interface{}

func parse(r io.Reader, vars varMap) (varMap, []byte, error) {
	type parserState int
	const (
		parseVar parserState = iota
		parseTitle
		parseBody
	)

	br := bufio.NewReader(r)
	out := &bytes.Buffer{}
	if vars == nil {
		vars = varMap{}
	}
	state := parseVar
	for {
		line, err := br.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}
	byState:
		switch state {
		case parseVar:
			cmd := strings.TrimRight(string(line), " \t\n")
			if cmd != "" && !strings.HasPrefix(cmd, "@") {
				state = parseTitle
				goto byState
			}
			if cmd == "" {
				break
			}
			pair := spaces.Split(cmd[1:], 2)
			if len(pair) != 2 {
				return nil, nil, fmt.Errorf("Invalid command format %s", cmd)
			}
			vars[pair[0]] = pair[1]
		case parseTitle:
			title := strings.TrimRight(string(line), " \t\n")
			if !strings.HasPrefix(title, "# ") {
				state = parseBody
				goto byState
			}
			out.Write(line)
			if _, exists := vars["title"]; !exists {
				vars["title"] = strings.TrimSpace(title[2:])
			}
			state = parseBody
		case parseBody:
			out.Write(line)
			if _, err := io.Copy(out, br); err != nil {
				return nil, nil, err
			}
		}
	}
	return vars, out.Bytes(), nil
}

func generatePage(tpl *template.Template, srcFn string) error {
	dstFn := srcFn[:len(srcFn)-len(filepath.Ext(srcFn))] + ".html"
	src, err := os.Open(srcFn)
	if err != nil {
		return err
	}
	defer src.Close()

	vars, mdContent, err := parse(src, nil)
	if err != nil {
		return err
	}
	vars["content"] = template.HTML(string(blackfriday.Run(mdContent)))

	dst, err := os.OpenFile(dstFn, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0640)
	if err != nil {
		return err
	}
	defer dst.Close()

	return tpl.Execute(dst, vars)
}

func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		log.Fatal("Usage: gensite basedir")
	}
	basedir := flag.Arg(0)

	tpl, err := template.ParseFiles(filepath.Join(basedir, "template.html"))
	if err != nil {
		log.Fatalf("Cannot load template.html from %s, %v", basedir, err)
	}

	filepath.Walk(basedir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Cannot handle %s, %v, skipped", path, err)
			return nil
		}
		if info.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		if err := generatePage(tpl, path); err != nil {
			log.Printf("Generate page failed %s, %v, skipped", path, err)
			return nil
		}
		return nil
	})
}
