// © Ben Garrett https://github.com/bengarrett/dupers

// Dupers is the blazing-fast file duplicate checker and filename search.
package main

import (
	"errors"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/bengarrett/dupers/dupers"
	"github.com/bengarrett/dupers/out"
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
	winOS = "windows"

	perfMsg                 = "To improve performance use the quiet flag"
	winRemind time.Duration = 10 * time.Second
)

type tasks struct {
	debug    *bool
	exact    *bool
	filename *bool
	lookup   *bool
	quiet    *bool
	rm       *bool
	rmPlus   *bool
	sensen   *bool
}

func flags(t *tasks) {
	t.exact = flag.Bool("exact", false, "match case")
	t.debug = flag.Bool("debug", false, "debug mode") // hidden flag
	t.filename = flag.Bool("name", false, "search for filenames, and ignore directories")
	t.lookup = flag.Bool("fast", false, "query the database for a much faster match,"+
		"\n\t\tthe results maybe stale as it does not look for any file changes on your system")
	t.quiet = flag.Bool("quiet", false, "quiet mode hides all but essential feedback"+
		"\n\tthis improves performance with slow, default terminal programs")
	t.sensen = flag.Bool("sensen", false, "delete everything in the <directory to check>;"+
		"\n\t\texcept for directories containing unique Windows programs and assets")
	t.rm = flag.Bool("delete", false, "delete the duplicate files found in the <directory to check>")
	t.rmPlus = flag.Bool("delete+", false, "delete the duplicate files and remove empty directories from the <directory to check>")
}

func main() {
	c, t := dupers.Config{}, tasks{}
	c.SetTimer()
	flags(&t)
	ex := flag.Bool("e", false, "alias for exact")
	f := flag.Bool("f", false, "alias for fast")
	fn := flag.Bool("n", false, "alias for name")
	h := flag.Bool("h", false, "alias for help")
	q := flag.Bool("q", false, "alias for quiet")
	v := flag.Bool("v", false, "alias for version")
	hlp := flag.Bool("help", false, "print help")
	ver := flag.Bool("version", false, "version and information for this program")
	flag.Usage = func() {
		help()
	}
	flag.Parse()
	if *h || *hlp {
		fmt.Print(help())
		return
	}
	if *q || *t.quiet {
		*t.quiet = true
		c.Quiet = true
	}
	if *t.debug {
		c.Debug = true
	}
	if s := options(ver, v); s != "" {
		fmt.Print(s)
		return
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
		if *f {
			*t.lookup = true
		}
		dupeCmd(&c, t, flag.Args()...)
	case "search":
		if *ex {
			t.exact = ex
		}
		if *fn {
			t.filename = fn
		}
		searchCmd(t, flag.Args()...)
	default:
		defaultCmd(selection)
	}
}

func defaultCmd(selection string) {
	out.ErrCont(ErrCmd)
	fmt.Printf("Command: '%s'\n\nSee the help for the available commands and options:\n", selection)
	out.Example("dupers --help")
	out.ErrFatal(nil)
}

func options(ver, v *bool) string {
	// convenience for when a help or version flag is passed as an argument
	for _, arg := range flag.Args() {
		switch strings.ToLower(arg) {
		case "-h", "-help", "--help":
			return help()
		case "-v", "-version", "--version":
			return info()
		}
	}
	// print version information
	if *ver || *v {
		return info()
	}
	// print help if no arguments are given
	if len(flag.Args()) == 0 {
		return help()
	}
	return ""
}
