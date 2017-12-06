package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/jroimartin/gocui"
	"github.com/y-yagi/configure"
)

type config struct {
	Languages []string `toml:"languages"`
}

type repository struct {
	name string
	lang string
	desc string
}

var reposPerLang map[string][]repository = map[string][]repository{}

func main() {
	os.Exit(run(os.Args, os.Stdout, os.Stderr))
}

func run(args []string, outStream, errStream io.Writer) (exitCode int) {
	var cfg config
	var editConfig bool
	exitCode = 0

	flags := flag.NewFlagSet("github-trending", flag.ExitOnError)
	flags.SetOutput(errStream)
	flags.BoolVar(&editConfig, "c", false, "configure")
	flags.Parse(args[1:])

	err := configure.Load("github-trending", &cfg)
	if err != nil {
		fmt.Fprintf(errStream, "%v\n", err)
		exitCode = 1
		return
	}

	if editConfig {
		editor := os.Getenv("EDITOR")
		if len(editor) == 0 {
			editor = "vim"
		}

		if err := configure.Edit("github-trending", editor); err != nil {
			fmt.Fprintf(outStream, "%v\n", err)
			exitCode = 1
			return
		}

		return
	}

	if len(cfg.Languages) == 0 {
		cfg.Languages = append(cfg.Languages, "")
	}

	for _, lang := range cfg.Languages {
		if err := fetchTrending(lang); err != nil {
			fmt.Fprintf(errStream, "%v\n", err)
			exitCode = 1
			return
		}
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
	var repos []repository

	key := language
	if len(key) == 0 {
		key = "all"
	}

	doc.Find("ol.repo-list li").Each(func(i int, s *goquery.Selection) {
		name := strings.TrimSpace(s.Find("h3").Text())
		repo.name = strings.Replace(name, " ", "", -1)
		repo.desc = strings.TrimSpace(s.Find(".py-1").Text())
		repo.lang = s.Find("[itemprop=programmingLanguage]").Text()
		repos = append(repos, repo)
	})

	reposPerLang[key] = repos

	return nil
}

func keybindings(g *gocui.Gui) error {
	if err := g.SetKeybinding("", gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyArrowLeft, gocui.ModNone, cursorLeft); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("", gocui.KeyArrowRight, gocui.ModNone, cursorRight); err != nil {
		log.Panicln(err)
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
	var firstKey string

	maxX, maxY := g.Size()
	if v, err := g.SetView("side", -1, -1, int(0.2*float32(maxX)), maxY); err != nil {
		v.Title = "Language"
		v.Highlight = true
		v.SelBgColor = gocui.ColorBlue
		v.SelFgColor = gocui.ColorBlack

		for k, _ := range reposPerLang {
			if len(firstKey) == 0 {
				firstKey = k
			}
			fmt.Fprintln(v, k)
		}
	}

	if v, err := g.SetView("main", int(0.2*float32(maxX)), -1, maxX, maxY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "GitHub Trending"
		v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack

		for _, r := range reposPerLang[firstKey] {
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
		if v, err = g.SetCurrentView("main"); err != nil {
			return err
		}
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
	var err error

	if v == nil {
		if v, err = g.SetCurrentView("main"); err != nil {
			return err
		}
	}

	cx, cy := v.Cursor()
	if err := v.SetCursor(cx, cy+1); err != nil {
		ox, oy := v.Origin()
		if err := v.SetOrigin(ox, oy+1); err != nil {
			return err
		}
	}

	if v.Name() == "side" {
		refreshMainView(g, v)
	}

	return nil
}

func cursorUp(g *gocui.Gui, v *gocui.View) error {
	var err error

	if v == nil {
		if v, err = g.SetCurrentView("main"); err != nil {
			return err
		}
	}

	ox, oy := v.Origin()
	cx, cy := v.Cursor()
	if err := v.SetCursor(cx, cy-1); err != nil && oy > 0 {
		if err := v.SetOrigin(ox, oy-1); err != nil {
			return err
		}
	}

	if v.Name() == "side" {
		refreshMainView(g, v)
	}

	return nil
}

func cursorLeft(g *gocui.Gui, v *gocui.View) error {
	var err error
	if v, err = g.SetCurrentView("side"); err != nil {
		return err
	}

	cx, cy := v.Cursor()
	if err := v.SetCursor(cx, cy); err != nil {
		ox, oy := v.Origin()
		if err := v.SetOrigin(ox, oy); err != nil {
			return err
		}
	}
	return nil
}

func cursorRight(g *gocui.Gui, v *gocui.View) error {
	var err error
	if v, err = g.SetCurrentView("main"); err != nil {
		fmt.Printf("%v\n", err)
		return err
	}

	cx, cy := v.Cursor()
	if err := v.SetCursor(cx, cy); err != nil {
		ox, oy := v.Origin()
		if err := v.SetOrigin(ox, oy); err != nil {
			return err
		}
	}
	return nil
}

func refreshMainView(g *gocui.Gui, v *gocui.View) error {
	var l string
	var err error

	mainView, _ := g.View("main")
	_, cy := v.Cursor()

	if l, err = v.Line(cy); err != nil {
		l = ""
	}

	if len(l) != 0 {
		mainView.Clear()
		for _, r := range reposPerLang[l] {
			fmt.Fprintln(mainView, "["+r.name+"] "+r.desc)
		}
	}
	return nil
}
