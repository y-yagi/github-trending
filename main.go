package main

import (
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/jroimartin/gocui"
)

type repository struct {
	name string
	lang string
	desc string
}

var repos []repository

func main() {
	os.Exit(run(os.Args, os.Stdout, os.Stderr))
}

func run(args []string, outStream, errStream io.Writer) (exitCode int) {
	var language string
	exitCode = 0

	flags := flag.NewFlagSet("github-trending", flag.ExitOnError)
	flags.SetOutput(errStream)
	flags.StringVar(&language, "l", "", "Language. Default: All")
	flags.Parse(args[1:])

	url := "https://github.com/trending/" + url.QueryEscape(language)
	doc, err := goquery.NewDocument(url)
	if err != nil {
		fmt.Fprintf(errStream, "URL get error: %v\n", err)
		exitCode = 1
		return
	}

	var repo repository

	doc.Find("ol.repo-list li").Each(func(i int, s *goquery.Selection) {
		name := strings.TrimSpace(s.Find("h3").Text())
		repo.name = strings.Replace(name, " ", "", -1)
		repo.desc = strings.TrimSpace(s.Find(".py-1").Text())
		repo.lang = s.Find("[itemprop=programmingLanguage]").Text()
		repos = append(repos, repo)
	})

	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		fmt.Fprintf(errStream, "GUI create error: %v\n", err)
		exitCode = 1
		return
	}
	defer g.Close()

	g.Cursor = true
	g.SetManagerFunc(layout)

	if err := keybindings(g); err != nil {
		fmt.Fprintf(errStream, "Key bindings error: %v\n", err)
		exitCode = 1
		return
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		fmt.Fprintf(errStream, "Unexpected error: %v\n", err)
		exitCode = 1
		return
	}

	return
}
