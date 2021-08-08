// © Ben Garrett https://github.com/bengarrett/dupers

// Dupers is the blazing-fast file duplicate checker and filename search.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bengarrett/dupers/dupers"
	"github.com/bengarrett/dupers/out"
	"github.com/gookit/color"
)

var (
	version = "0.0.0"
	commit  = "unset" // nolint: gochecknoglobals
	date    = "unset" // nolint: gochecknoglobals
)

var (
	ErrCmd          = errors.New("command is unknown")
	ErrDatabaseName = errors.New("database has no bucket name")
	ErrImport       = errors.New("import filepath is missing")
	ErrNewName      = errors.New("a new directory is required")
	ErrNoArgs       = errors.New("request is missing arguments")
	ErrSearch       = errors.New("search request needs an expression")
	ErrWindowsDir   = errors.New("cannot parse the directory path")
)

const (
	dbf   = "database"
	dbs   = "db"
	dbk   = "backup"
	dcn   = "clean"
	dex   = "export"
	dim   = "import"
	dls   = "ls"
	dmv   = "mv"
	drm   = "rm"
	dup   = "up"
	dupp  = "up+"
	fhlp  = "-help"
	winOS = "windows"

	perfMsg                 = "To improve performance use the quiet flag"
	winRemind time.Duration = 10 * time.Second
)

type tasks struct {
	debug    *bool
	exact    *bool
	filename *bool
	help     *bool
	lookup   *bool
	mono     *bool
	quiet    *bool
	rm       *bool
	rmPlus   *bool
	sensen   *bool
	version  *bool
}

type aliases struct {
	exact    *bool
	filename *bool
	help     *bool
	lookup   *bool
	mono     *bool
	quiet    *bool
	version  *bool
}

func flags(t *tasks) {
	t.exact = flag.Bool("exact", false, "match case")
	t.debug = flag.Bool("debug", false, "debug mode") // hidden flag
	t.filename = flag.Bool("name", false, "search for filenames, and ignore directories")
	t.help = flag.Bool("help", false, "print help") // only used in certain circumstances
	t.lookup = flag.Bool("fast", false, "query the database for a much faster match,"+
		"\n\t\tthe results maybe stale as it does not look for any file changes on your system")
	t.mono = flag.Bool("mono", false, "monochrome mode to remove all color output")
	t.quiet = flag.Bool("quiet", false, "quiet mode hides all but essential feedback"+
		"\n\tthis improves performance with slow, default terminal programs")
	t.sensen = flag.Bool("sensen", false, "delete everything in the <directory to check>;"+
		"\n\t\texcept for directories containing unique Windows programs and assets")
	t.rm = flag.Bool("delete", false, "delete the duplicate files found in the <directory to check>")
	t.rmPlus = flag.Bool("delete+", false, "delete the duplicate files and remove empty directories from the <directory to check>")
	t.version = flag.Bool("version", false, "version and information for this program")
}

func shortFlags(a *aliases) {
	a.exact = flag.Bool("e", false, "alias for exact")
	a.lookup = flag.Bool("f", false, "alias for fast")
	a.filename = flag.Bool("n", false, "alias for name")
	a.help = flag.Bool("h", false, "alias for help")
	a.mono = flag.Bool("m", false, "alias for mono")
	a.quiet = flag.Bool("q", false, "alias for quiet")
	a.version = flag.Bool("v", false, "alias for version")
}

func parse(a *aliases, c *dupers.Config, t *tasks) (exit bool) {
	if *a.mono || *t.mono {
		color.Enable = false
	}
	if *a.help || *t.help {
		fmt.Print(help())
		return true
	}
	if *a.quiet || *t.quiet {
		*t.quiet = true
		c.Quiet = true
	}
	if *t.debug {
		c.Debug = true
	}
	if *a.lookup {
		*t.lookup = true
	}
	if *a.exact {
		t.exact = a.exact
	}
	if *a.filename {
		t.filename = a.filename
	}
	if s := options(t.version, a.version); s != "" {
		fmt.Print(s)
		return true
	}
	return false
}

func main() {
	a, c, t := aliases{}, dupers.Config{}, tasks{}
	c.SetTimer()
	flags(&t)
	shortFlags(&a)
	flag.Usage = func() {
		help()
	}
	flag.Parse()
	if parse(&a, &c, &t) {
		os.Exit(0)
	}
	chkWinDirs()
	selection := strings.ToLower(flag.Args()[0])
	if c.Debug {
		out.Bug("command selection: " + selection)
	}
	switch selection {
	case dbf, dbs, dbk, dcn, dex, dim, dls, dmv, drm, dup, dupp:
		databaseCmd(&c, *t.quiet, flag.Args()...)
	case "dupe":
		dupeCmd(&c, &t, flag.Args()...)
	case "search":
		searchCmd(&t, flag.Args()...)
	default:
		defaultCmd(selection)
	}
}

func defaultCmd(selection string) {
	out.ErrCont(ErrCmd)
	fmt.Printf("Command: '%s'\n\nSee the help for the available commands and options:\n", selection)
	out.Example("dupers " + fhlp)
	out.ErrFatal(nil)
}

func options(ver, v *bool) string {
	// convenience for when a flag is passed as an argument
	for _, arg := range flag.Args() {
		switch strings.ToLower(arg) {
		case "-h", fhlp, "--help":
			return help()
		case "-v", "-version", "--version":
			return vers()
		}
	}
	if *ver || *v {
		return vers()
	}
	if len(flag.Args()) == 0 {
		return help()
	}
	return ""
}
