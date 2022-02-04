// © Ben Garrett https://github.com/bengarrett/dupers

// Dupers is a blazing-fast file duplicate checker and filename search tool.
package main

import (
	"bytes"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/bengarrett/dupers/database"
	"github.com/bengarrett/dupers/dupe"
	"github.com/bengarrett/dupers/internal/cmd"
	"github.com/bengarrett/dupers/internal/out"
	"github.com/bengarrett/dupers/internal/task"
	"github.com/gookit/color"
)

var ErrCmd = errors.New("command is unknown")

// logo.txt by sensenstahl
//go:embed logo.txt
var brand string

var (
	version = "0.0.0"
	commit  = "unset" // nolint: gochecknoglobals
	date    = "unset" // nolint: gochecknoglobals
)

const (
	dbf  = "database"
	dbs  = "db"
	dbk  = "backup"
	dcn  = "clean"
	dex  = "export"
	dim  = "import"
	dls  = "ls"
	dmv  = "mv"
	drm  = "rm"
	dup  = "up"
	dupp = "up+"
	fhlp = "-help"
)

func main() {
	a, c, f := cmd.Aliases{}, dupe.Config{}, cmd.Flags{}
	c.SetTimer()
	cmd.Define(&f)
	cmd.DefineShort(&a)
	flag.Usage = func() {
		task.Help()
	}
	flag.Parse()
	parse(&a, &c, &f)
	if err := task.ChkWinDirs(); err != nil {
		out.ErrFatal(err)
	}
	selection := strings.ToLower(flag.Args()[0])
	if c.Debug {
		out.PBug("command selection: " + selection)
	}

	switch selection {
	case "dupe":
		if err := task.Dupe(&c, &f, flag.Args()...); err != nil {
			out.ErrFatal(err)
		}
	case "search":
		task.Search(&f, flag.Args()...)
	case dbf, dbs, dbk, dcn, dex, dim, dls, dmv, drm, dup, dupp:
		if err := task.Database(&c, *f.Quiet, flag.Args()...); err != nil {
			if errors.Is(err, database.ErrDBNotFound) {
				os.Exit(0)
			}
			if errors.Is(err, database.ErrDBZeroByte) {
				os.Exit(1)
			}
			out.ErrFatal(err)
		}
	default:
		defaultCmd(selection)
	}
}

// parse the command aliases and flags and returns true if the program should exit.
func parse(a *cmd.Aliases, c *dupe.Config, f *cmd.Flags) {
	if *a.Mono || *f.Mono {
		color.Enable = false
	}
	if s := options(a, f); s != "" {
		fmt.Printf("%s", s)
		os.Exit(0)
	}
	if *f.Debug {
		c.Debug = true
	}
	if *a.Quiet || *f.Quiet {
		*f.Quiet = true
		c.Quiet = true
	}
	if *a.Exact {
		*f.Exact = true
	}
	if *a.Filename {
		*f.Filename = true
	}
	if *a.Lookup {
		*f.Lookup = true
	}
}

func defaultCmd(selection string) {
	out.ErrCont(ErrCmd)
	fmt.Printf("Command: '%s'\n\nSee the help for the available commands and options:\n", selection)
	out.Example("dupers " + fhlp)
	out.ErrFatal(nil)
}

// options parses universal aliases, flags and any misuse.
func options(a *cmd.Aliases, f *cmd.Flags) string {
	if *a.Help || *f.Help {
		return task.Help()
	}
	// handle misuse when a flag is passed as an argument
	for _, arg := range flag.Args() {
		switch strings.ToLower(arg) {
		case "-h", fhlp, "--help":
			return task.Help()
		case "-v", "-version", "--version":
			return vers()
		}
	}
	if *a.Version || *f.Version {
		return vers()
	}
	if len(flag.Args()) == 0 {
		return task.Help()
	}
	return ""
}

// vers prints out the program information and version.
func vers() string {
	const copyright, year = "\u00A9", "2021-22"
	exe, err := cmd.Self()
	if err != nil {
		out.ErrCont(err)
	}
	w := new(bytes.Buffer)
	fmt.Fprintln(w, brand+"\n")
	fmt.Fprintf(w, "                                dupers v%s\n", version)
	fmt.Fprintf(w, "                        %s %s Ben Garrett\n", copyright, year)
	fmt.Fprintf(w, "         %s\n\n", color.Primary.Sprint("https://github.com/bengarrett/dupers"))
	fmt.Fprintf(w, "  %s    %s (%s)\n", color.Secondary.Sprint("build:"), commit, date)
	fmt.Fprintf(w, "  %s %s/%s\n", color.Secondary.Sprint("platform:"), runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(w, "  %s       %s\n", color.Secondary.Sprint("go:"), strings.Replace(runtime.Version(), "go", "v", 1))
	fmt.Fprintf(w, "  %s     %s\n", color.Secondary.Sprint("path:"), exe)
	return w.String()
}
