package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/y-yagi/configure"
	"github.com/y-yagi/gocui"
	"github.com/y-yagi/goext/osext"
)

type config struct {
	Languages []string `toml:"languages"`
	Browser   string   `toml:"browser"`
}

type repository struct {
	name     string
	desc     string
	language string
	stars    string
}

var reposPerLang = map[string][]repository{}
var cfg config
var lang string

const (
	mainView  = "main"
	sideView  = "side"
	appName   = "github-trending"
	githubURL = "https://github.com/"
)

func init() {
	f := filepath.Join(configure.ConfigDir(appName), "config.toml")
	if !osext.IsExist(f) {
		c := config{Languages: []string{"all"}, Browser: "google-chrome"}
		configure.Save(appName, c)
	}
}

func main() {
	os.Exit(run(os.Args, os.Stdout, os.Stderr))
}

func run(args []string, outStream, errStream io.Writer) (exitCode int) {
	var configureFlag bool
	exitCode = 0

	flags := flag.NewFlagSet(appName, flag.ExitOnError)
	flags.SetOutput(errStream)
	flags.BoolVar(&configureFlag, "c", false, "configure")
	flags.Parse(args[1:])

	err := configure.Load(appName, &cfg)
	if err != nil {
		fmt.Fprintf(errStream, "%v\n", err)
		exitCode = 1
		return
	}

	if configureFlag {
		if err = editConfig(); err != nil {
			fmt.Fprintf(outStream, "%v\n", err)
			exitCode = 1
		}
		return
	}

	if len(cfg.Languages) == 0 {
		fmt.Fprintln(outStream, "Please specify Languages.")
		return
	}

	var wg sync.WaitGroup
	for _, lang := range cfg.Languages {
		wg.Add(1)
		go fetchTrending(lang, errStream, &wg)
	}
	wg.Wait()

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

func editConfig() error {
	editor := os.Getenv("EDITOR")
	if len(editor) == 0 {
		editor = "vim"
	}

	return configure.Edit(appName, editor)
}

func fetchTrending(language string, errStream io.Writer, wg *sync.WaitGroup) {
	defer wg.Done()
	u := githubURL + "trending/"

	if language != "all" {
		u += url.QueryEscape(language)
	}

	doc, err := goquery.NewDocument(u)
	if err != nil {
		fmt.Fprintf(errStream, "%v\n", err)
		return
	}

	var repo repository
	var repos []repository

	doc.Find("article.Box-row").Each(func(i int, s *goquery.Selection) {
		h1 := s.Find("h1").Text()
		repoName := strings.Split(h1, "/")
		repo.name = fmt.Sprintf("%s/%s", strings.TrimSpace(repoName[0]), strings.TrimSpace(repoName[1]))
		repo.desc = strings.TrimSpace(s.Find("p.pr-4").Text())
		repo.language = strings.TrimSpace(s.Find("span[itemprop='programmingLanguage']").Text())
		repo.stars = strings.TrimSpace(s.Find("a.mr-3").First().Text())
		repos = append(repos, repo)
	})

	reposPerLang[language] = repos

	return
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
	if err := g.SetKeybinding("", 'j', gocui.ModNone, cursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'k', gocui.ModNone, cursorUp); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'h', gocui.ModNone, cursorLeft); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'q', gocui.ModNone, quit); err != nil {
		return err
	}

	return g.SetKeybinding("", 'l', gocui.ModNone, cursorRight)
}

func layout(g *gocui.Gui) error {

	maxX, maxY := g.Size()
	if v, err := g.SetView(sideView, -1, 0, int(0.2*float32(maxX)), maxY); err != nil {
		v.Title = "Language"
		v.Highlight = true
		v.SelBgColor = gocui.ColorBlue
		v.SelFgColor = gocui.ColorBlack

		for k := range reposPerLang {
			if len(lang) == 0 {
				lang = k
			}
			fmt.Fprintln(v, k)
		}
	}

	if v, err := g.SetView(mainView, int(0.2*float32(maxX)), 0, maxX, int(0.8*float32(maxY))); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "GitHub Trending"
		v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack

		for _, r := range reposPerLang[lang] {
			fmt.Fprintln(v, "["+r.name+"] "+r.desc)
		}
	}

	// show some details
	if v, err := g.SetView("details", int(0.2*float32(maxX)), int(0.8*float32(maxY)), maxX, maxY); err != nil {

		if err != gocui.ErrUnknownView {
			return err
		}

		v.Title = "Details"
		v.Highlight = false
		v.SelFgColor = gocui.ColorBlack
		v.Wrap = true
		refreshDetailsView(g)
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
		if v, err = g.SetCurrentView(mainView); err != nil {
			return err
		}
	}

	_, cy := v.Cursor()
	if l, err = v.Line(cy); err != nil {
		l = ""
	}

	repo := strings.TrimLeft(strings.Split(l, "]")[0], "[")
	url := githubURL + repo
	if err := exec.Command(cfg.Browser, url).Run(); err != nil {
		if err := openByDefault(url); err != nil {
			fmt.Printf("%v\n", err)
			return err
		}
	}

	return nil
}

func openByDefault(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default:
		cmd = "xdg-open"
	}

	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func cursorDown(g *gocui.Gui, v *gocui.View) error {
	var err error

	if v == nil {
		if v, err = g.SetCurrentView(mainView); err != nil {
			return err
		}
	}

	cx, cy := v.Cursor()
	lineCount := len(strings.Split(v.ViewBuffer(), "\n"))
	if cy+1 == lineCount-2 {
		return nil
	}
	if err := v.SetCursor(cx, cy+1); err != nil {
		ox, oy := v.Origin()
		if err := v.SetOrigin(ox, oy+1); err != nil {
			return err
		}
	}

	return drawInfoViews(g, v)
}

func cursorUp(g *gocui.Gui, v *gocui.View) error {
	var err error

	if v == nil {
		if v, err = g.SetCurrentView(mainView); err != nil {
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
	return drawInfoViews(g, v)
}

func drawInfoViews(g *gocui.Gui, v *gocui.View) error {
	var err error

	if v.Name() == sideView {
		// set the language which is used in both main and details view
		setLang(g, v)
		if err = refreshMainView(g, v); err != nil {
			return err
		}

		if err = refreshDetailsView(g); err != nil {
			return err
		}

	}

	if v.Name() == mainView {
		if err = refreshDetailsView(g); err != nil {
			return err
		}
	}

	return nil
}

func setLang(g *gocui.Gui, v *gocui.View) error {

	var l string
	var err error

	if v.Name() == sideView {
		_, cy := v.Cursor()

		if l, err = v.Line(cy); err != nil {
			l = ""
		}

		lang = l
	}
	return nil
}

func cursorLeft(g *gocui.Gui, v *gocui.View) error {
	var err error
	if v, err = g.SetCurrentView(sideView); err != nil {
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
	if v, err = g.SetCurrentView(mainView); err != nil {
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
	return refreshDetailsView(g)
}

func refreshDetailsView(g *gocui.Gui) error {
	mainView, _ := g.View(mainView)
	_, cy := mainView.Cursor()

	detailsView, _ := g.View("details")
	detailsView.Clear()

	repo := reposPerLang[lang][cy]

	fmt.Fprintf(detailsView, "%s 🌟%s\n", repo.language, repo.stars)
	fmt.Fprintf(detailsView, "%s", repo.desc)
	return nil
}

func refreshMainView(g *gocui.Gui, v *gocui.View) error {
	var l string
	var err error

	mainView, _ := g.View(mainView)
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
