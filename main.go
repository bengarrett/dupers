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
	author   = "Ben Garrett"
	homepage = "https://github.com/bengarrett/dupers"
)

func main() {
	a, c, f := cmd.Aliases{}, dupe.Config{}, cmd.Flags{}
	c.SetTimer()
	f.Define()
	a.Define()
	flag.Usage = func() {
		task.Help()
	}
	flag.Parse()

	c = f.Aliases(&a, &c)
	if *a.Mono || *f.Mono {
		color.Enable = false
	}

	if help := exitOptions(&a, &f); help != "" {
		fmt.Fprint(os.Stdout, help)
		os.Exit(0)
	}

	if err := task.Directories(); err != nil {
		out.ErrFatal(err)
	}

	selection := strings.ToLower(flag.Args()[0])
	if c.Debug {
		out.PBug("command selection: " + selection)
	}
	switch selection {
	case task.Dupe_:
		const testing = false
		if err := task.Dupe(&c, &f, testing, flag.Args()...); err != nil {
			out.ErrFatal(err)
		}
	case task.Search_:
		if err := task.Search(&f, false, flag.Args()...); err != nil {
			out.ErrFatal(err)
		}
	case
		task.Database_,
		task.DB_,
		task.Backup_,
		task.Clean_,
		task.Export_,
		task.Import_,
		task.LS_,
		task.MV_,
		task.RM_,
		task.Up_,
		task.UpPlus_:
		if err := task.Database(&c, *f.Yes, *f.Quiet, flag.Args()...); err != nil {
			if errors.Is(err, database.ErrDBNotFound) {
				os.Exit(0)
			}
			if errors.Is(err, database.ErrDBZeroByte) {
				os.Exit(1)
			}
			out.ErrFatal(err)
		}
	default:
		unknown(selection)
	}
}

// unknown returns a command is unknown helper error.
func unknown(s string) {
	w := os.Stderr
	out.ErrCont(ErrCmd)
	fmt.Fprintf(w, "Command: '%s'", s)
	fmt.Fprintln(w)
	fmt.Fprintln(w)
	fmt.Fprint(w, "See the help for the available commands and options:")
	fmt.Fprintln(w)
	out.Example("dupers -help")

	os.Exit(1)
}

// exitOptions parses help and version options.
func exitOptions(a *cmd.Aliases, f *cmd.Flags) string {
	noArgs := len(flag.Args()) == 0
	if *f.Version && *f.Debug {
		return task.Debug(a, f)
	}
	if *a.Help || *f.Help {
		selection := ""
		if !noArgs {
			selection = strings.ToLower(flag.Args()[0])
		}
		switch selection {
		case task.Dupe_:
			return task.HelpDupe()
		case task.Search_:
			return task.HelpSearch()
		case task.Database_, task.DB_:
			return task.HelpDatabase()
		default:
			return task.Help()
		}
	}
	if *a.Version || *f.Version {
		return about(*f.Quiet)
	}
	// no commands or arguments, then always return help
	if noArgs {
		return task.Help()
	}
	return ""
}

// about returns the program branding and information.
func about(quiet bool) string {
	const width = 45
	exe, err := cmd.Self()
	if err != nil {
		out.ErrCont(err)
	}
	w := new(bytes.Buffer)
	if !quiet {
		fmt.Fprintln(w, brand+"\n")
		fmt.Fprintln(w, ralign(width, "dupers v"+version))
		fmt.Fprintln(w, ralign(width, fmt.Sprintf("%s %s", copyright(), author)))
		fmt.Fprintln(w, color.Primary.Sprint(ralign(width, homepage)))
		fmt.Fprintln(w)
	}
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
