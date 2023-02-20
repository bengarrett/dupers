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

	"github.com/bengarrett/dupers/internal/printer"
	"github.com/bengarrett/dupers/pkg/cmd"
	"github.com/bengarrett/dupers/pkg/cmd/task/bucket"
	"github.com/bengarrett/dupers/pkg/cmd/task/duplicate"
	"github.com/bengarrett/dupers/pkg/cmd/task/search"
	"github.com/bengarrett/dupers/pkg/database"
	"github.com/bengarrett/dupers/pkg/dupe"
	"github.com/dustin/go-humanize"
	"github.com/gookit/color"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	ErrArgs     = errors.New("no buckets were given as arguments")
	ErrCommand  = errors.New("command is unknown")
	ErrNilFlags = errors.New("flags cannot be a nil value")
	ErrNoArgs   = errors.New("arguments cannot be empty")
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
	const minArgs = 2
	if len(flag.Args()) < minArgs {
		return nil
	}
	var errs error
	directories := flag.Args()[1:]
	for _, dir := range directories {
		if err := cmd.WindowsChk(dir); err != nil {
			errs = errors.Join(errs, err)
			continue
		}
	}
	if errs != nil {
		return errs
	}
	return nil
}

// Database parses the commands that interact with the database.
func Database(db *bolt.DB, c *dupe.Config, args ...string) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	if c == nil {
		return dupe.ErrNilConfig
	}
	if len(args) == 0 {
		return ErrCommand
	}
	buckets := [2]string{}
	copy(buckets[:], args)
	quiet := c.Quiet
	assumeYes := c.Yes
	if _, err := database.Check(); err != nil {
		return err
	}
	selection := args[0]
	switch selection {
	case Backup_:
		return backupDB(quiet)
	case Clean_:
		return CleanupDB(db, c)
	case DB_, Database_:
		s, err := database.Info(db)
		if err != nil {
			printer.StderrCR(err)
		}
		fmt.Fprintln(os.Stdout, s)
	case Export_:
		return bucket.Export(db, quiet, buckets)
	case Import_:
		return bucket.Import(db, quiet, assumeYes, buckets)
	case LS_:
		return bucket.List(db, quiet, buckets)
	case MV_:
		return bucket.Move(db, c, assumeYes, buckets)
	case RM_:
		return bucket.Remove(db, quiet, assumeYes, buckets)
	case Up_:
		return bucket.Rescan(db, c, false, buckets)
	case UpPlus_:
		return bucket.Rescan(db, c, true, buckets)
	default:
		return ErrCommand
	}
	return nil
}

// Dupe parses the dupe command.
func Dupe(db *bolt.DB, c *dupe.Config, f *cmd.Flags, testing bool, args ...string) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	if f == nil {
		return ErrNilFlags
	}
	if f.Version == nil {
		return ErrNilFlags
	}
	c.Debugger(fmt.Sprintf("dupe command: %s", strings.Join(args, " ")))

	// fetch bucket info
	b, err := database.All(db)
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
	if err := WalkCheck(db, c, args...); err != nil {
		return err
	}
	return WalkScan(db, c, f, args...)
}

// WalkCheck checks directories and files to scan, a bucket is the name given to database tables.
func WalkCheck(db *bolt.DB, c *dupe.Config, args ...string) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	if c == nil {
		return dupe.ErrNilConfig
	}
	const minArgs = 2
	if len(args) < minArgs {
		return ErrNoArgs
	}
	buckets := args[2:]
	if len(buckets) == 0 {
		if err := c.SetAllBuckets(db); err != nil {
			return err
		}
		c.Debugger(fmt.Sprintf("use all buckets: %s", c.BucketS()))
		return nil
	}
	if err := c.SetBuckets(buckets...); err != nil {
		return err
	}
	if err := CheckDupePaths(c); err != nil {
		return err
	}
	c.Debugger(fmt.Sprintf("use buckets: %s", c.BucketS()))
	return nil
}

func WalkScan(db *bolt.DB, c *dupe.Config, f *cmd.Flags, args ...string) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	if c == nil {
		return dupe.ErrNilConfig
	}
	// files or directories to compare (these are not saved to database)
	if err := c.WalkSource(); err != nil {
		return err
	}
	c.Debugger("walksource complete.")
	// walk, scan and save file paths and hashes to the database
	if err := duplicate.WalkScanSave(db, c, f); err != nil {
		return err
	}
	if !c.Quiet {
		fmt.Fprint(os.Stdout, printer.EraseLine())
	}
	// print the found dupes
	s, err := c.Print()
	if err != nil {
		return err
	}
	fmt.Fprint(os.Stdout, s)
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
	ProgramOpts(w)
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
	ProgramOpts(w)
	if err := w.Flush(); err != nil {
		return fmt.Sprintf("could not flush the help text: %s", err)
	}
	return b.String()
}

func HelpDupe() string {
	b, w := helper()
	DupeHelp(w)
	ProgramOpts(w)
	if err := w.Flush(); err != nil {
		return fmt.Sprintf("could not flush the help text: %s", err)
	}
	return b.String()
}

func HelpSearch() string {
	b, w := helper()
	SearchHelp(w)
	ProgramOpts(w)
	if err := w.Flush(); err != nil {
		return fmt.Sprintf("could not flush the help text: %s", err)
	}
	return b.String()
}

// Search parses the commands that handle search.
func Search(db *bolt.DB, f *cmd.Flags, test bool, args ...string) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	if f == nil {
		return ErrNilFlags
	}
	if f.Filename == nil || f.Exact == nil || f.Quiet == nil {
		return ErrNilFlags
	}
	l := len(args)
	if l == 0 {
		return ErrArgs
	}
	if err := search.CmdErr(l, test); err != nil {
		return err
	}
	term, buckets := args[1], []string{}
	const minArgs = 2
	if l > minArgs {
		buckets = args[minArgs:]
	}
	m, err := search.Compare(db, f, term, buckets, false)
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
	name, writ, err := database.Backup()
	if err != nil {
		return err
	}
	if quiet {
		fmt.Fprintln(os.Stdout, name)
	}
	s := fmt.Sprintf("A new copy of the database (%s) is at: %s",
		humanize.Bytes(uint64(writ)), name)
	printer.Quiet(quiet, s)
	return nil
}

// cleanupDB cleans and compacts the database.
func CleanupDB(db *bolt.DB, c *dupe.Config) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	if c == nil {
		return dupe.ErrNilConfig
	}
	if err := database.Clean(db, c.Quiet, c.Debug); err != nil {
		if b := errors.Is(err, database.ErrNoClean); !b {
			return err
		}
		printer.StderrCR(err)
	}
	if err := database.Compact(db, c.Debug); err != nil {
		if b := errors.Is(err, database.ErrNoCompact); !b {
			return err
		}
	}
	return nil
}

// CheckDupePaths checks the path arguments supplied to the dupe command.
func CheckDupePaths(c *dupe.Config) error {
	if c == nil {
		return dupe.ErrNilConfig
	}
	files, buckets, err := c.CheckPaths()
	if err != nil {
		return err
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
	if buckets == 0 {
		fmt.Fprintf(w, " %s ", c.GetSource())
	} else {
		fmt.Fprintf(w, " %s ", color.Warn.Sprint(c.GetSource()))
	}
	fmt.Fprintf(w, "(%s)", color.Info.Sprintf("%s files", p.Sprint(files)))
	fmt.Fprintln(w)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s to lookup, for finding duplicates:", verb)
	fmt.Fprintln(w)
	fmt.Fprintf(w, " %s ", c.BucketS())
	if buckets == 0 {
		fmt.Fprintf(w, "(%s)", color.Danger.Sprintf("%s files", p.Sprint(buckets)))
		fmt.Fprintln(w)
		fmt.Fprintln(w)
		fmt.Fprintln(os.Stderr, color.Danger.Sprintf("The %s to lookup contains no files", strings.ToLower(verb)))
		return bucket.ErrBucketEmpty
	}
	fmt.Fprintf(w, "(%s)", color.Info.Sprintf("%s files", p.Sprint(buckets)))
	fmt.Fprintln(w)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "The bucket to lookup is to be stored in the database,")
	color.Warn.Println(" but the \"Directory to check\" is not.")
	if !printer.AskYN("Is this what you want", c.Yes, printer.No) {
		return nil
	}
	return nil
}
