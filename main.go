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
	"time"
	"unicode/utf8"

	"github.com/bengarrett/dupers/database"
	"github.com/bengarrett/dupers/dupe"
	"github.com/bengarrett/dupers/internal/cmd"
	"github.com/bengarrett/dupers/internal/out"
	"github.com/bengarrett/dupers/internal/task"
	"github.com/carlmjohnson/versioninfo"
	"github.com/gookit/color"
)

var ErrCmd = errors.New("command is unknown")

// logo.txt by sensenstahl
//
//go:embed internal/logo.txt
var brand string

// version only gets updated when built using GoReleaser.
var version = "0.0.0"

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

	homepage = "https://github.com/bengarrett/dupers"
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
		const testing = false
		if err := task.Dupe(&c, &f, testing, flag.Args()...); err != nil {
			out.ErrFatal(err)
		}
	case "search":
		if err := task.Search(&f, false, flag.Args()...); err != nil {
			out.ErrFatal(err)
		}
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
		fmt.Fprintf(os.Stdout, "%s", s)
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
	fmt.Fprintf(os.Stdout, "Command: '%s'\n\nSee the help for the available commands and options:\n", selection)
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
	const width = 45
	exe, err := cmd.Self()
	if err != nil {
		out.ErrCont(err)
	}
	w := new(bytes.Buffer)
	fmt.Fprintln(w, brand+"\n")
	fmt.Fprintln(w, ralign(width, "dupers v"+version))
	fmt.Fprintln(w, ralign(width, copyright()+" Ben Garrett"))
	fmt.Fprintln(w, color.Primary.Sprint(ralign(width, homepage)))
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %s    %s\n", color.Secondary.Sprint("build:"), commit())
	fmt.Fprintf(w, "  %s %s/%s\n", color.Secondary.Sprint("platform:"),
		runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(w, "  %s       %s\n", color.Secondary.Sprint("go:"),
		strings.Replace(runtime.Version(), "go", "v", 1))
	fmt.Fprintf(w, "  %s     %s\n", color.Secondary.Sprint("path:"), exe)
	return w.String()
}

// ralign aligns the string to the right of the terminal using space padding.
func ralign(maxWidth int, s string) string {
	l := utf8.RuneCountInString(s)
	if l >= maxWidth {
		return s
	}
	diff := maxWidth - l
	return fmt.Sprintf("%s%s", strings.Repeat(" ", diff), s)
}

// copyright year range for this program.
func copyright() string {
	const initYear = 2021
	t := versioninfo.LastCommit
	s := fmt.Sprintf("© %d-", initYear)
	if t.Year() > initYear {
		s += t.Local().Format("06")
	} else {
		s += time.Now().Format("06")
	}
	return s
}

// commit returns a formatted, git commit description for this repository,
// including tag version and date.
func commit() string {
	s := ""
	c := versioninfo.Short()

	if c == "devel" {
		return c
	}
	if c != "" {
		s += fmt.Sprintf("%s, ", c)
	}
	if l := lastCommit(); l != "" {
		s += fmt.Sprintf("built on %s", l)
	}
	if s == "" {
		return "n/a"
	}
	return strings.TrimSpace(s)
}

func lastCommit() string {
	d := versioninfo.LastCommit
	if d.IsZero() {
		return ""
	}
	return d.Local().Format("2006 Jan 2 15:04")
}
