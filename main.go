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

	winRemind time.Duration = 10 * time.Second
)

// cmdFlags are options for commands.
type cmdFlags struct {
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

// aliases are single letter options for commands.
type aliases struct {
	exact    *bool
	filename *bool
	help     *bool
	lookup   *bool
	mono     *bool
	quiet    *bool
	version  *bool
}

// flags defines options for the commands.
func flags(f *cmdFlags) {
	f.exact = flag.Bool("exact", false, "match case")
	f.debug = flag.Bool("debug", false, "debug mode") // hidden flag
	f.filename = flag.Bool("name", false, "search for filenames, and ignore directories")
	f.help = flag.Bool("help", false, "print help") // only used in certain circumstances
	f.lookup = flag.Bool("fast", false, "query the database for a much faster match,"+
		"\n\t\tthe results maybe stale as it does not look for any file changes on your system")
	f.mono = flag.Bool("mono", false, "monochrome mode to remove all color output")
	f.quiet = flag.Bool("quiet", false, "quiet mode hides all but essential feedback"+
		"\n\tthis improves performance with slow, default terminal programs")
	f.sensen = flag.Bool("sensen", false, "delete everything in the <directory to check>;"+
		"\n\t\texcept for directories containing unique Windows programs and assets")
	f.rm = flag.Bool("delete", false, "delete the duplicate files found in the <directory to check>")
	f.rmPlus = flag.Bool("delete+", false, "delete the duplicate files and remove empty directories from the <directory to check>")
	f.version = flag.Bool("version", false, "version and information for this program")
}

// shortFlags defines options for the command aliases.
func shortFlags(a *aliases) {
	a.exact = flag.Bool("e", false, "alias for exact")
	a.lookup = flag.Bool("f", false, "alias for fast")
	a.filename = flag.Bool("n", false, "alias for name")
	a.help = flag.Bool("h", false, "alias for help")
	a.mono = flag.Bool("m", false, "alias for mono")
	a.quiet = flag.Bool("q", false, "alias for quiet")
	a.version = flag.Bool("v", false, "alias for version")
}

// parse the command aliases and flags and returns true if the program should exit.
func parse(a *aliases, c *dupers.Config, f *cmdFlags) (exit bool) {
	if *a.mono || *f.mono {
		color.Enable = false
	}
	if s := options(a, f); s != "" {
		fmt.Print(s)
		return true
	}
	if *f.debug {
		c.Debug = true
	}
	if *a.quiet || *f.quiet {
		*f.quiet = true
		c.Quiet = true
	}
	if *a.exact {
		*f.exact = true
	}
	if *a.filename {
		*f.filename = true
	}
	if *a.lookup {
		*f.lookup = true
	}
	return false
}

func main() {
	a, c, f := aliases{}, dupers.Config{}, cmdFlags{}
	c.SetTimer()
	flags(&f)
	shortFlags(&a)
	flag.Usage = func() {
		help()
	}
	flag.Parse()
	if parse(&a, &c, &f) {
		os.Exit(0)
	}
	chkWinDirs()
	selection := strings.ToLower(flag.Args()[0])
	if c.Debug {
		out.Bug("command selection: " + selection)
	}
	switch selection {
	case "dupe":
		dupeCmd(&c, &f, flag.Args()...)
	case "search":
		searchCmd(&f, flag.Args()...)
	case dbf, dbs, dbk, dcn, dex, dim, dls, dmv, drm, dup, dupp:
		databaseCmd(&c, *f.quiet, flag.Args()...)
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

// options parses universal aliases, flags and any misuse.
func options(a *aliases, f *cmdFlags) string {
	if *a.help || *f.help {
		return help()
	}
	// handle misuse when a flag is passed as an argument
	for _, arg := range flag.Args() {
		switch strings.ToLower(arg) {
		case "-h", fhlp, "--help":
			return help()
		case "-v", "-version", "--version":
			return vers()
		}
	}
	if *a.version || *f.version {
		return vers()
	}
	if len(flag.Args()) == 0 {
		return help()
	}
	return ""
}
