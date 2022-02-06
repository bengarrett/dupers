package cmd

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/bengarrett/dupers/internal/out"
	"github.com/gookit/color"
)

var ErrWindowsDir = errors.New("cannot parse the directory path")

// Aliases are single letter options for commands.
type Aliases struct {
	Exact    *bool
	Filename *bool
	Help     *bool
	Lookup   *bool
	Mono     *bool
	Quiet    *bool
	Version  *bool
}

// Flags are options for commands.
type Flags struct {
	Debug    *bool
	Exact    *bool
	Filename *bool
	Help     *bool
	Lookup   *bool
	Mono     *bool
	Quiet    *bool
	Rm       *bool
	RmPlus   *bool
	Sensen   *bool
	Version  *bool
}

// ChkWinDir checks the string for invalid escaped quoted paths when using using Windows cmd.exe.
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

// Define options for the commands.
func Define(f *Flags) {
	if f == nil {
		return
	}
	f.Exact = flag.Bool("exact", false, "match case")
	f.Debug = flag.Bool("debug", false, "debug mode") // hidden flag
	f.Filename = flag.Bool("name", false, "search for filenames, and ignore directories")
	f.Help = flag.Bool("help", false, "print help") // only used in certain circumstances
	f.Lookup = flag.Bool("fast", false, "query the database for a much faster match,"+
		"\n\t\tthe results maybe stale as it does not look for any file changes on your system")
	f.Mono = flag.Bool("mono", false, "monochrome mode to remove all color output")
	f.Quiet = flag.Bool("quiet", false, "quiet mode hides all but essential feedback")
	f.Sensen = flag.Bool("sensen", false, "delete everything in the <directory to check>"+
		"\n\t\texcept for directories containing unique Windows programs and assets")
	f.Rm = flag.Bool("delete", false, "delete the duplicate files found in the <directory to check>")
	f.RmPlus = flag.Bool("delete+", false,
		"delete the duplicate files and remove empty directories from the <directory to check>")
	f.Version = flag.Bool("version", false, "version and information for this program")
}

// DefineShort options for the command aliases.
func DefineShort(a *Aliases) {
	if a == nil {
		return
	}
	a.Exact = flag.Bool("e", false, "alias for exact")
	a.Lookup = flag.Bool("f", false, "alias for fast")
	a.Filename = flag.Bool("n", false, "alias for name")
	a.Help = flag.Bool("h", false, "alias for help")
	a.Mono = flag.Bool("m", false, "alias for mono")
	a.Quiet = flag.Bool("q", false, "alias for quiet")
	a.Version = flag.Bool("v", false, "alias for version")
}

// Home returns the user's home directory.
// Or if that fails, returns the current working directory.
func Home() string {
	h, err := os.UserHomeDir()
	if err != nil {
		if h, err = os.Getwd(); err != nil {
			out.ErrCont(err)
		}
	}
	return h
}

// Self returns the path to this dupers executable file.
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
		return fmt.Sprintf("%s%s exist for '%s'.", t, color.Secondary.Sprint(s), color.Bold.Sprint(term))
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
