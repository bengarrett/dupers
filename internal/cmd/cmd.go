// © Ben Garrett https://github.com/bengarrett/dupers
package cmd

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/bengarrett/dupers/dupe"
	"github.com/bengarrett/dupers/internal/out"
	"github.com/gookit/color"
)

var ErrWindowsDir = errors.New("cannot parse the directory path")

const (
	Debug_   = "debug"
	Delete_  = "delete"
	DelPlus_ = "delete+"
	Exact_   = "exact"
	Fast_    = "fast"
	Help_    = "help"
	Mono_    = "mono"
	Name_    = "name"
	Quiet_   = "quiet"
	Sensen_  = "sensen"
	Yes_     = "yes"
	Version_ = "version"
)

// Aliases are single letter options for commands.
type Aliases struct {
	Debug    *bool `usage:"alias for debug"`
	Exact    *bool `usage:"alias for exact"`
	Filename *bool `usage:"alias for filename"`
	Help     *bool `usage:"alias for help"`
	Lookup   *bool `usage:"alias for lookup"`
	Mono     *bool `usage:"alias for mono"`
	Quiet    *bool `usage:"alias for quiet"`
	Yes      *bool `usage:"alias for quiet"`
	Version  *bool `usage:"alias for version"`
}

// Usage of the command aliases.
func (a *Aliases) Usage(name string) string {
	t := reflect.TypeOf(*a)
	sf, ok := t.FieldByName(name)
	if !ok {
		return ""
	}
	val, ok := sf.Tag.Lookup("usage")
	if !ok {
		return ""
	}
	return val
}

// Define optional aliases for the program and commands flags.
func (a *Aliases) Define() {
	if a == nil {
		return
	}
	a.Debug = flag.Bool("d", false, a.Usage("Debug"))
	a.Exact = flag.Bool("e", false, a.Usage("Exact"))
	a.Lookup = flag.Bool("f", false, a.Usage("Lookup"))
	a.Filename = flag.Bool("n", false, a.Usage("Filename"))
	a.Help = flag.Bool("h", false, a.Usage("Help"))
	a.Mono = flag.Bool("m", false, a.Usage("Mono"))
	a.Quiet = flag.Bool("q", false, a.Usage("Quiet"))
	a.Yes = flag.Bool("y", false, a.Usage("Yes"))
	a.Version = flag.Bool("v", false, a.Usage("Version"))
}

// Flags provide options for both the commands and the program.
type Flags struct {
	Exact    *bool `usage:"match case"`
	Filename *bool `usage:"search for filenames, and ignore directories"`
	Lookup   *bool `usage:"query the database for a much faster match, the results\n\t maybe stale as it does not look for any file changes on\n\t your system"`
	Rm       *bool `usage:"delete everything in the <directory to check> except\n\t for directories containing unique Windows programs and\n\t assets"`
	RmPlus   *bool `usage:"delete the duplicate files found in the\n\t <directory to check>"`
	Sensen   *bool `usage:"delete the duplicate files and remove empty directories\n\t from the <directory to check>"`

	// global options

	Debug   *bool `usage:"debug is a verbose mode to print all the activities\n\t and tasks"`
	Help    *bool `usage:"print help"`
	Mono    *bool `usage:"monochrome mode to remove all color output"`
	Quiet   *bool `usage:"quiet mode hides all but essential feedback"`
	Yes     *bool `usage:"assume yes for any user prompts"`
	Version *bool `usage:"version and information for this program"`
}

// Usage of the command flags.
func (f *Flags) Usage(name string) string {
	t := reflect.TypeOf(*f)
	sf, ok := t.FieldByName(name)
	if !ok {
		return ""
	}
	val, ok := sf.Tag.Lookup("usage")
	if !ok {
		return ""
	}
	return val
}

// Define options for the program and commands.
func (f *Flags) Define() {
	if f == nil {
		return
	}
	f.Debug = flag.Bool(Debug_, false, f.Usage("Debug"))
	f.Exact = flag.Bool(Exact_, false, f.Usage("Exact"))
	f.Filename = flag.Bool(Name_, false, f.Usage("Filename"))
	f.Help = flag.Bool(Help_, false, f.Usage("Help")) // only used in certain circumstances
	f.Lookup = flag.Bool(Fast_, false, f.Usage("Lookup"))
	f.Mono = flag.Bool(Mono_, false, f.Usage("Mono"))
	f.Quiet = flag.Bool(Quiet_, false, f.Usage("Quiet"))
	f.Sensen = flag.Bool(Sensen_, false, f.Usage("Sensen"))
	f.Rm = flag.Bool(Delete_, false, f.Usage("Rm"))
	f.RmPlus = flag.Bool(DelPlus_, false, f.Usage("RmPlus"))
	f.Yes = flag.Bool(Yes_, false, f.Usage("Yes"))
	f.Version = flag.Bool(Version_, false, f.Usage("Version"))
}

// Aliases parses the command aliases and flags, configuring both Flags and dupe.Config.
func (f *Flags) Aliases(a *Aliases, c *dupe.Config) dupe.Config {
	// handle misuse when a global flag is passed as an argument
	for _, arg := range flag.Args() {
		switch strings.ToLower(arg) {
		case "-d", "-debug", "--debug":
			*f.Debug = true
		case "-m", "-mono", "--mono":
			*f.Mono = true
		case "-q", "-quiet", "--quiet":
			*f.Quiet = true
		case "-h", "-help", "--help":
			*f.Help = true
		case "-y", "-yes", "--yes":
			*f.Yes = true
		case "-v", "-version", "--version":
			*f.Version = true
		default:
			// help and version are handled by main.suffixOpts()
		}
	}
	// configurations
	if *a.Debug || *f.Debug {
		*f.Debug = true
		c.Debug = true
	}
	if *a.Exact {
		*f.Exact = true
	}
	if *a.Filename {
		*f.Filename = true
	}
	if *a.Help {
		*f.Help = true
	}
	if *a.Lookup {
		*f.Lookup = true
	}
	if *a.Mono {
		*f.Mono = true
	}
	if *a.Quiet || *f.Quiet {
		*f.Quiet = true
		c.Quiet = true
	}
	if *a.Yes {
		*f.Yes = true
	}
	if *a.Version {
		*f.Version = true
	}
	return *c
}

// ChkWinDir checks the string for invalid, escaped quoted paths when using Windows cmd.exe.
func ChkWinDir(s string) error {
	if s == "" {
		return nil
	}
	const dblQuote rune = 34
	r := []rune(s)
	l := len(r)
	first, last := r[0:1][0], r[l-1 : l][0]
	if first == dblQuote && last == dblQuote {
		return nil // okay as the string is fully quoted
	}
	if first != dblQuote && last != dblQuote {
		return nil // okay as the string is not quoted
	}
	// otherwise there is a problem, as only the start or end of the string is quoted.
	// this is caused by flag.Parse() treating the \" prefix on a quoted directory path as an escaped quote.
	// so "C:\Example\" will be incorrectly parsed as C:\Example"
	w := new(bytes.Buffer)
	fmt.Fprint(w, "please remove the trailing backslash \\ character from any quoted directory paths")
	if usr, err := os.UserHomeDir(); err == nil {
		fmt.Fprint(w, "\n")
		fmt.Fprint(w, color.Success.Sprint("Good: "))
		fmt.Fprintf(w, "\"%s\" ", usr)
		fmt.Fprint(w, "\n")
		fmt.Fprint(w, color.Warn.Sprint("Bad: "))
		fmt.Fprintf(w, "\"%s\\\"", usr)
	}
	return fmt.Errorf("%w\n%s", ErrWindowsDir, w.String())
}

// Home returns the user home directory.
// If that fails it returns the current working directory.
func Home() string {
	h, err := os.UserHomeDir()
	if err != nil {
		if h, err = os.Getwd(); err != nil {
			out.ErrCont(err)
		}
	}
	return h
}

// Self returns the path to the dupers executable file.
func Self() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("self error: %w", err)
	}
	return exe, nil
}

// SearchSummary formats the results of the search command.
func SearchSummary(total int, term string, exact, filename bool) string {
	str := func(t, s, term string) string {
		return fmt.Sprintf("%s%s exist for '%s'.", t,
			color.Secondary.Sprint(s), color.Bold.Sprint(term))
	}
	s, r := "", "results"
	if total == 0 {
		return fmt.Sprintf("No results exist for '%s'.", term)
	}
	if total == 1 {
		r = "result"
	}
	t := color.Primary.Sprint(total)
	if exact && filename {
		s += fmt.Sprintf(" exact filename %s", r)
		return str(t, s, term)
	}
	if exact {
		s += fmt.Sprintf(" exact %s", r)
		return str(t, s, term)
	}
	if filename {
		s += fmt.Sprintf(" filename %s", r)
		return str(t, s, term)
	}
	s += fmt.Sprintf(" %s", r)
	return str(t, s, term)
}
