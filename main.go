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

	"github.com/bengarrett/dupers/internal/printer"
	"github.com/bengarrett/dupers/pkg/cmd"
	"github.com/bengarrett/dupers/pkg/cmd/task"
	"github.com/bengarrett/dupers/pkg/database"
	"github.com/bengarrett/dupers/pkg/dupe"
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

func tasks(selection string, a cmd.Aliases, c *dupe.Config, f cmd.Flags) error {
	switch selection {
	case task.Dupe_:
		db, err := database.OpenRead()
		if err != nil {
			return err
		}
		defer db.Close()
		return task.Dupe(db, c, &f, false, flag.Args()...)
	case task.Search_:
		db, err := database.OpenRead()
		if err != nil {
			return err
		}
		defer db.Close()
		return task.Search(db, &f, false, flag.Args()...)
	case
		task.Backup_,
		task.Clean_,
		task.Database_, task.DB_,
		task.Export_,
		task.LS_:
		db, err := database.OpenRead()
		if err != nil {
			return err
		}
		defer db.Close()
		return task.Database(db, c, flag.Args()...)
	case
		task.Import_,
		task.MV_,
		task.RM_,
		task.Up_,
		task.UpPlus_:
		db, err := database.OpenWrite()
		if err != nil {
			return err
		}
		defer db.Close()
		return task.Database(db, c, flag.Args()...)
	default:
		unknownExit(selection)
	}
	return fmt.Errorf("%w: %q", ErrCmd, selection)
}

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

	help, err := taskHelpVer(&a, &f)
	if err != nil {
		printer.ErrFatal(err)
	}
	if help != "" {
		fmt.Fprint(os.Stdout, help)
		os.Exit(0)
	}

	if err := task.Directories(); err != nil {
		printer.ErrFatal(err)
	}

	selection := strings.ToLower(flag.Args()[0])
	c.DPrint("command selection: " + selection)
	if err := tasks(selection, a, &c, f); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			os.Exit(0)
		}
		if errors.Is(err, database.ErrZeroByte) {
			os.Exit(1)
		}
		printer.ErrFatal(err)
	}
}

// unknownExit prints the command is unknown helper error and exits.
func unknownExit(s string) {
	w := os.Stderr
	printer.StderrCR(ErrCmd)
	fmt.Fprintf(w, "Command: '%s'", s)
	fmt.Fprintln(w)
	fmt.Fprintln(w)
	fmt.Fprint(w, "See the help for the available commands and options:")
	fmt.Fprintln(w)
	printer.Example("dupers -help")

	os.Exit(1)
}

// taskHelpVer returns the help or version options.
func taskHelpVer(a *cmd.Aliases, f *cmd.Flags) (string, error) {
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
			return task.HelpDupe(), nil
		case task.Search_:
			return task.HelpSearch(), nil
		case task.Database_, task.DB_:
			return task.HelpDatabase(), nil
		default:
			return task.Help(), nil
		}
	}
	if *a.Version || *f.Version {
		return about(*f.Quiet), nil
	}
	// no commands or arguments, then always return help
	if noArgs {
		return task.Help(), nil
	}
	return "", nil
}

// about returns the program branding and information.
func about(quiet bool) string {
	const width = 45
	exe, err := cmd.Self()
	if err != nil {
		printer.StderrCR(err)
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
