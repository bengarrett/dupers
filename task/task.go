// © Ben Garrett https://github.com/bengarrett/dupers

// Package task prepares and run commands for the dupers package.
package task

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"text/tabwriter"

	dupe "github.com/bengarrett/dupers"
	"github.com/bengarrett/dupers/database"
	"github.com/bengarrett/dupers/internal/cmd"
	"github.com/bengarrett/dupers/internal/out"
	"github.com/bengarrett/dupers/task/internal/bucket"
	"github.com/bengarrett/dupers/task/internal/duplicate"
	"github.com/bengarrett/dupers/task/internal/help"
	"github.com/bengarrett/dupers/task/internal/search"
	"github.com/dustin/go-humanize"
	"github.com/gookit/color"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	ErrArgs   = errors.New("no buckets were given as arguments")
	ErrCfg    = errors.New("config cannot be a nil value")
	ErrCmd    = errors.New("command is unknown")
	ErrFast   = errors.New("the fast method cannot be used in this situation")
	ErrFlag   = errors.New("flags cannot be a nil value")
	ErrNil    = errors.New("argument cannot be a nil value")
	ErrSyntax = errors.New("command line arguments syntax")
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
)

// ChkWinDirs checks the arguments for invalid escaped quoted paths when using using Windows cmd.exe.
func ChkWinDirs() error {
	if runtime.GOOS != winOS {
		return nil
	}
	if len(flag.Args()) > 1 {
		for _, s := range flag.Args()[1:] {
			if err := cmd.ChkWinDir(s); err != nil {
				return err
			}
		}
	}
	return nil
}

// Database parses the commands that interact with the database.
func Database(c *dupe.Config, quiet bool, args ...string) error {
	if err := database.Check(""); err != nil {
		return err
	}
	if len(args) == 0 {
		return ErrCmd
	}
	buckets := [2]string{}
	copy(buckets[:], args)
	switch args[0] {
	case dbk:
		return backupDB(quiet)
	case dcn:
		return cleanupDB(quiet, c.Debug)
	case dbs, dbf:
		s, err := database.Info("")
		if err != nil {
			out.ErrCont(err)
		}
		fmt.Println(s)
	case dex:
		bucket.Export(quiet, buckets)
	case dim:
		bucket.Import(quiet, buckets)
	case dls:
		bucket.List(quiet, buckets)
	case dmv:
		buckets := [3]string{}
		copy(buckets[:], args)
		bucket.Move(quiet, buckets)
	case drm:
		bucket.Remove(quiet, buckets)
	case dup:
		bucket.Rescan(c, false, buckets)
	case dupp:
		bucket.Rescan(c, true, buckets)
	default:
		return ErrCmd
	}
	return nil
}

// Dupe parses the dupe command.
func Dupe(c *dupe.Config, f *cmd.Flags, args ...string) error { // nolint:cyclop,funlen
	if c == nil {
		return ErrCfg
	}
	if f == nil {
		return ErrFlag
	}
	if c.Debug {
		s := fmt.Sprintf("dupeCmd: %s", strings.Join(args, " "))
		out.DebugLn(s)
	}
	l := len(args)
	if l == 1 {
		const minArgs = 2
		fmt.Print(duplicate.CmdErr(l, 0, minArgs))
		return ErrSyntax
	}
	// fetch bucket info
	b, err := database.All(nil)
	if err != nil {
		return err
	}
	const minArgs = 3
	if l < minArgs && len(b) == 0 {
		fmt.Print(duplicate.CmdErr(l, len(b), minArgs))
		return ErrSyntax
	}
	// directory or a file to match
	const minReq = 2
	if len(args) < minReq {
		return ErrArgs
	}
	c.SetToCheck(args[1])
	// directories and files to scan, a bucket is the name given to database tables
	if buckets := args[2:]; len(buckets) == 0 {
		if err := c.SetBuckets(); err != nil {
			return err
		}
	} else {
		c.SetBucket(buckets...)
		checkDupePaths(c)
	}
	if c.Debug {
		s := fmt.Sprintf("buckets: %s", c.PrintBuckets())
		out.DebugLn(s)
	}
	// files or directories to compare (these are not saved to database)
	if err := c.WalkSource(); err != nil {
		return err
	}
	if c.Debug {
		out.DebugLn("walksource complete.")
	}
	// walk, scan and save file paths and hashes to the database
	Lookup(c, *f.Lookup)
	if !c.Quiet {
		fmt.Print(out.RMLine())
	}
	// print the found dupes
	fmt.Print(c.Print())
	// remove files
	duplicate.Cleanup(c, f)
	// summaries
	if !c.Quiet {
		fmt.Println(c.Status())
	}
	return nil
}

// Help, usage and examples.
func Help() string {
	const (
		tabPadding  = 4
		description = "Dupers is the blazing-fast file duplicate checker and filename search tool."
	)
	b, f := bytes.Buffer{}, flag.Flag{}
	w := tabwriter.NewWriter(&b, 0, 0, tabPadding, ' ', 0)

	fmt.Fprintf(w, "%s\n", description)

	help.Dupe(f, w)
	help.Search(f, w)
	help.DB(f, w)
	fmt.Fprintln(w)
	if err := w.Flush(); err != nil {
		return ""
	}
	return b.String()
}

// Lookup both cleans and then updates the buckets with file system changes.
// The fast bool queries the database instead of the file system for a quicker match.
func Lookup(c *dupe.Config, fast bool) {
	if c.Debug {
		out.DebugLn("dupe lookup.")
	}
	// normalise bucket names
	for i, b := range c.All() {
		abs, err := database.Abs(string(b))
		if err != nil {
			out.ErrCont(err)
			c.All()[i] = ""
			continue
		}
		c.All()[i] = dupe.Bucket(abs)
	}
	buckets := make([]string, 0, len(c.All()))
	for _, b := range c.All() {
		buckets = append(buckets, string(b))
	}
	if !fast && len(buckets) > 0 {
		if c.Debug {
			out.DebugLn("non-fast mode, database cleanup.")
		}
		if err := database.Clean(c.Quiet, c.Debug, buckets...); err != nil {
			out.ErrCont(err)
		}
	}
	if fast {
		if err := lookup(c); err != nil {
			return
		}
	}
	if c.Debug {
		out.DebugLn("walk the buckets.")
	}
	c.WalkDirs()
}

func lookup(c *dupe.Config) error {
	if c.Debug {
		out.DebugLn("read the hash values in the buckets.")
	}
	fastErr := false
	for _, b := range c.All() {
		if i, err := c.SetCompares(b); err != nil {
			out.ErrCont(err)
		} else if i > 0 {
			continue
		}
		fastErr = true
		fmt.Println("The -fast flag cannot be used for this dupe query")
	}
	if !fastErr {
		return ErrFast
	}
	return nil
}

// Searchr parses the commands that handle search.
func Searchr(f *cmd.Flags, test bool, args ...string) error {
	if f == nil {
		return ErrFlag
	}
	l := len(args)
	if err := search.CmdErr(l, test); err != nil {
		return err
	}

	s := Search{
		Term:     args[1],
		Filename: *f.Filename,
		Exact:    *f.Exact,
		Test:     test,
		Buckets:  nil,
	}
	const minArgs = 2
	if l > minArgs {
		s.Buckets = args[minArgs:]
	}
	m, err := s.Compare()
	if err != nil {
		return err
	}

	fmt.Print(dupe.Print(*f.Quiet, *f.Exact, s.Term, m))
	if !*f.Quiet {
		l := 0
		if m != nil {
			l = len(*m)
		}
		fmt.Println(cmd.SearchSummary(l, s.Term, *f.Exact, *f.Filename))
	}
	return nil
}

// backupDB saves the database to a binary file.
func backupDB(quiet bool) error {
	n, w, err := database.Backup()
	if err != nil {
		return err
	}
	s := fmt.Sprintf("A new copy of the database (%s) is at: %s", humanize.Bytes(uint64(w)), n)
	out.Response(s, quiet)
	return nil
}

// cleanupDB cleans and compacts the database.
func cleanupDB(quiet, debug bool) error {
	if err := database.Clean(quiet, debug); err != nil {
		if b := errors.Is(err, database.ErrDBClean); !b {
			return err
		}
		out.ErrCont(err)
	}
	if err := database.Compact(debug); err != nil {
		if b := errors.Is(err, database.ErrDBCompact); !b {
			return err
		}
	}
	return nil
}

// checkDupePaths checks the path arguments supplied to the dupe command.
func checkDupePaths(c *dupe.Config) {
	if c == nil {
		return
	}
	ok, cc, bc := c.CheckPaths()
	if ok {
		return
	}
	p := message.NewPrinter(language.English)
	verb := "Buckets"
	if len(c.All()) == 1 {
		verb = "Bucket"
	}
	fmt.Printf("Directory to check:\n %s (%s)\n", c.ToCheck(), color.Info.Sprintf("%s files", p.Sprint(cc)))
	fmt.Printf("%s to lookup, for finding duplicates:\n %s (%s)\n\n",
		verb, c.PrintBuckets(), color.Info.Sprintf("%s files", p.Sprint(bc)))
	color.Warn.Println("\"Directory to check\" is NOT saved to the database.")
	if !out.YN("Is this what you want", out.No) {
		os.Exit(0)
	}
}

// Search request to find a file or directory name within the buckets.
type Search struct {
	Term     string   // Term to match, either a partial or complete file or directory name.
	Filename bool     // Filename base name matches only, ignoring any directory paths.
	Exact    bool     // Exact case sensitive matching.
	Test     bool     // Test mode for unit tests.
	Buckets  []string // Buckets to search.
}

// Compare finds matches of the term within the buckets.
func (s Search) Compare() (*database.Matches, error) {
	var err error
	var m *database.Matches
	switch {
	case s.Filename && !s.Exact:
		if m, err = database.CompareBaseNoCase(s.Term, s.Buckets...); err != nil {
			return nil, search.Error(err, s.Test)
		}
	case s.Filename && s.Exact:
		if m, err = database.CompareBase(s.Term, s.Buckets...); err != nil {
			return nil, search.Error(err, s.Test)
		}
	case !s.Filename && !s.Exact:
		if m, err = database.CompareNoCase(s.Term, s.Buckets...); err != nil {
			return nil, search.Error(err, s.Test)
		}
	case !s.Filename && s.Exact:
		if m, err = database.Compare(s.Term, s.Buckets...); err != nil {
			return nil, search.Error(err, s.Test)
		}
	}
	return m, nil
}
