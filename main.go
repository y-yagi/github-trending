package main

import (
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
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

	if err := fetchTrending(language); err != nil {
		fmt.Fprintf(errStream, "%v\n", err)
		exitCode = 1
		return
	}

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

func fetchTrending(language string) error {
	url := "https://github.com/trending/" + url.QueryEscape(language)
	doc, err := goquery.NewDocument(url)
	if err != nil {
		return err
	}

	var repo repository

	doc.Find("ol.repo-list li").Each(func(i int, s *goquery.Selection) {
		name := strings.TrimSpace(s.Find("h3").Text())
		repo.name = strings.Replace(name, " ", "", -1)
		repo.desc = strings.TrimSpace(s.Find(".py-1").Text())
		repo.lang = s.Find("[itemprop=programmingLanguage]").Text()
		repos = append(repos, repo)
	})

	return nil
}

func keybindings(g *gocui.Gui) error {
	if err := g.SetKeybinding("", gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, open); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}
	return nil
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("main", int(0.2*float32(maxX)), -1, maxX, maxY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "GitHub Trending"
		v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack

		for _, r := range repos {
			fmt.Fprintln(v, "["+r.name+"] "+r.desc)
		}
	}
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func open(g *gocui.Gui, v *gocui.View) error {
	var l string
	var err error

	if v == nil {
		v = g.Views()[0]
	}

	_, cy := v.Cursor()
	if l, err = v.Line(cy); err != nil {
		l = ""
	}

	repo := strings.TrimLeft(strings.Split(l, "]")[0], "[")
	url := "https://github.com/" + repo
	return exec.Command("google-chrome", url).Run()
}

func cursorDown(g *gocui.Gui, v *gocui.View) error {
	if v == nil {
		v = g.Views()[0]
	}

	cx, cy := v.Cursor()
	if err := v.SetCursor(cx, cy+1); err != nil {
		ox, oy := v.Origin()
		if err := v.SetOrigin(ox, oy+1); err != nil {
			return err
		}
	}
	return nil
}

func cursorUp(g *gocui.Gui, v *gocui.View) error {
	if v == nil {
		v = g.Views()[0]
	}

	ox, oy := v.Origin()
	cx, cy := v.Cursor()
	if err := v.SetCursor(cx, cy-1); err != nil && oy > 0 {
		if err := v.SetOrigin(ox, oy-1); err != nil {
			return err
		}
	}
	return nil
}
