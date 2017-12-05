package main

import (
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	runewidth "github.com/mattn/go-runewidth"
	"github.com/olekukonko/tablewriter"
)

type repository struct {
	name string
	lang string
	desc string
}

func (r *repository) toArray() []string {
	return []string{
		r.name,
		runewidth.Truncate(r.desc, 80, "..."),
	}
}

func (r *repository) url() string {
	return "https://github.com/" + r.name
}

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
	table := tablewriter.NewWriter(outStream)
	table.SetColMinWidth(1, 100)

	doc.Find("ol.repo-list li").Each(func(i int, s *goquery.Selection) {
		name := strings.TrimSpace(s.Find("h3").Text())
		repo.name = strings.Replace(name, " ", "", -1)
		repo.desc = strings.TrimSpace(s.Find(".py-1").Text())
		repo.lang = s.Find("[itemprop=programmingLanguage]").Text()

		table.Append(repo.toArray())
	})

	table.Render()
	return
}
