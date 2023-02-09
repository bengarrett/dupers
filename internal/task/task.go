// © Ben Garrett https://github.com/bengarrett/dupers
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

	"github.com/bengarrett/dupers/database"
	"github.com/bengarrett/dupers/dupe"
	"github.com/bengarrett/dupers/internal/cmd"
	"github.com/bengarrett/dupers/internal/out"
	"github.com/bengarrett/dupers/internal/task/internal/bucket"
	"github.com/bengarrett/dupers/internal/task/internal/duplicate"
	"github.com/bengarrett/dupers/internal/task/internal/search"
	"github.com/dustin/go-humanize"
	"github.com/gookit/color"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	ErrArgs = errors.New("no buckets were given as arguments")
	ErrCfg  = errors.New("config cannot be a nil value")
	ErrCmd  = errors.New("command is unknown")
	ErrFlag = errors.New("flags cannot be a nil value")
	ErrNil  = errors.New("argument cannot be a nil value")
)

const (
	Backup_   = "backup"
	Clean_    = "clean"
	Database_ = "database"
	DB_       = "db"
	Dupe_     = "dupe"
	Export_   = "export"
	Import_   = "import"
	LS_       = "ls"
	MV_       = "mv"
	RM_       = "rm"
	Search_   = "search"
	Up_       = "up"
	UpPlus_   = "up+"
	winOS     = "windows"
)

const (
	tabPadding  = 4
	description = "Dupers is the blazing-fast file duplicate checker and filename search tool."
)

// Directories checks the arguments for invalid escaped quoted paths when using the Windows cmd.exe shell.
func Directories() error {
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
	if err := database.Check(); err != nil {
		return err
	}
	if len(args) == 0 {
		return ErrCmd
	}
	buckets := [2]string{}
	copy(buckets[:], args)
	switch args[0] {
	case Backup_:
		return backupDB(quiet)
	case Clean_:
		return cleanupDB(quiet, c.Debug)
	case DB_, Database_:
		s, err := database.Info()
		if err != nil {
			out.ErrCont(err)
		}
		fmt.Fprintln(os.Stdout, s)
	case Export_:
		bucket.Export(quiet, buckets)
	case Import_:
		bucket.Import(quiet, buckets)
	case LS_:
		bucket.List(quiet, buckets)
	case MV_:
		buckets := [3]string{}
		copy(buckets[:], args)
		bucket.Move(quiet, buckets)
	case RM_:
		bucket.Remove(quiet, buckets)
	case Up_:
		bucket.Rescan(c, false, buckets)
	case UpPlus_:
		bucket.Rescan(c, true, buckets)
	default:
		return ErrCmd
	}
	return nil
}

// Dupe parses the dupe command.
func Dupe(c *dupe.Config, f *cmd.Flags, testing bool, args ...string) error {
	if c == nil {
		return ErrCfg
	}
	if f == nil {
		return ErrFlag
	}
	c.DPrint(fmt.Sprintf("dupe command: %s", strings.Join(args, " ")))

	// fetch bucket info
	b, err := database.All(nil)
	if err != nil {
		return err
	}

	const minReq, source = 3, 1
	const minArgs = source + 1
	l := len(args)
	switch {
	case l == 1:
		duplicate.CmdErr(l, 0, minArgs, testing)
		return nil // TODO return err?
	case l < minArgs:
		if len(b) == 0 {
			duplicate.CmdErr(l, len(b), minArgs, testing)
			return nil // TODO return err?
		}
		return ErrArgs
	}
	if err := c.SetSource(args[source]); err != nil {
		return err
	}

	walkCheck(c, args...)
	return walkScan(c, f, args...)
}

// walkCheck checks directories and files to scan, a bucket is the name given to database tables.
func walkCheck(c *dupe.Config, args ...string) {
	s := fmt.Sprintf("buckets: %s", c.PrintBuckets())
	buckets := args[2:]
	if len(buckets) == 0 {
		if err := c.SetAllBuckets(); err != nil {
			out.ErrFatal(err)
		}
		if c.Debug {
			out.PBug(s)
		}
		return
	}
	c.SetBucket(buckets...)
	if code := checkDupePaths(c); code >= 0 {
		os.Exit(code)
	}
	if c.Debug {
		out.PBug(s)
	}
}

func walkScan(c *dupe.Config, f *cmd.Flags, args ...string) error {
	// files or directories to compare (these are not saved to database)
	if err := c.WalkSource(); err != nil {
		return err
	}
	if c.Debug {
		out.PBug("walksource complete.")
	}
	// walk, scan and save file paths and hashes to the database
	duplicate.Lookup(c, f)
	if !c.Quiet {
		fmt.Fprint(os.Stdout, out.RMLine())
	}
	// print the found dupes
	fmt.Fprint(os.Stdout, c.Print())
	// remove files
	duplicate.Cleanup(c, f)
	// summaries
	if !c.Quiet {
		fmt.Fprintln(os.Stdout, c.Status())
	}
	return nil
}

// Help, usage and examples.
func Help() string {
	b, w := helper()
	DupeHelp(w)
	SearchHelp(w)
	DatabaseHelp(w)
	fmt.Fprintln(w)
	if err := w.Flush(); err != nil {
		return fmt.Sprintf("could not flush the help text: %s", err)
	}
	return b.String()
}

func helper() (*bytes.Buffer, *tabwriter.Writer) {
	b := bytes.Buffer{}
	w := tabwriter.NewWriter(&b, 0, 0, tabPadding, ' ', 0)
	fmt.Fprintln(w, description)
	return &b, w
}

func HelpDatabase() string {
	b, w := helper()
	DatabaseHelp(w)
	if err := w.Flush(); err != nil {
		return fmt.Sprintf("could not flush the help text: %s", err)
	}
	return b.String()
}

func HelpDupe() string {
	b, w := helper()
	DupeHelp(w)
	if err := w.Flush(); err != nil {
		return fmt.Sprintf("could not flush the help text: %s", err)
	}
	return b.String()
}

func HelpSearch() string {
	b, w := helper()
	SearchHelp(w)
	if err := w.Flush(); err != nil {
		return fmt.Sprintf("could not flush the help text: %s", err)
	}
	return b.String()
}

// Search parses the commands that handle search.
func Search(f *cmd.Flags, test bool, args ...string) error {
	if f == nil {
		return ErrFlag
	}
	l := len(args)
	if err := search.CmdErr(l, test); err != nil {
		return err
	}
	term, buckets := args[1], []string{}
	const minArgs = 2
	if l > minArgs {
		buckets = args[minArgs:]
	}
	m, err := search.Compare(f, term, buckets, false)
	if err != nil {
		return err
	}
	fmt.Fprint(os.Stdout, dupe.Print(*f.Quiet, *f.Exact, term, m))
	if !*f.Quiet {
		l := 0
		if m != nil {
			l = len(*m)
		}
		fmt.Fprintln(os.Stdout, cmd.SearchSummary(l, term, *f.Exact, *f.Filename))
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
// An os.exit code is return or a -1 for no errors.
func checkDupePaths(c *dupe.Config) (code int) {
	if c == nil {
		return
	}
	files, buckets, err := c.CheckPaths()
	if err != nil {
		if errors.Is(err, dupe.ErrPathIsFile) {
			return -1
		}
		if errors.Is(err, os.ErrNotExist) {
			return 1
		}
		return 2
	}
	// handle any problems
	p := message.NewPrinter(language.English)
	verb := "Buckets"
	if len(c.All()) == 1 {
		verb = "Bucket"
	}
	w := os.Stdout
	fmt.Fprint(w, "Directory to check:")
	fmt.Fprintln(w)
	fmt.Fprintf(w, " %s ", c.ToCheck())
	fmt.Fprintf(w, "(%s)", color.Info.Sprintf("%s files", p.Sprint(files)))
	fmt.Fprintln(w)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s to lookup, for finding duplicates:", verb)
	fmt.Fprintln(w)
	fmt.Fprintf(w, " %s ", c.PrintBuckets())
	if buckets == 0 {
		fmt.Fprintf(w, "(%s)", color.Danger.Sprintf("%s files", p.Sprint(buckets)))
		fmt.Fprintln(w)
		fmt.Fprintln(w)
		fmt.Fprintln(os.Stderr, color.Danger.Sprintf("The %s to lookup contains no files", strings.ToLower(verb)))
		return 1
	}
	fmt.Fprintf(w, "(%s)", color.Info.Sprintf("%s files", p.Sprint(buckets)))
	fmt.Fprintln(w)
	fmt.Fprintln(w)
	color.Warn.Println("\"Directory to check\" is NOT saved to the database.")
	if !out.YN("Is this what you want", out.No) {
		return 0
	}
	return -1
}
