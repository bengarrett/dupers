// © Ben Garrett https://github.com/bengarrett/dupers
package task

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
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
	bberr "go.etcd.io/bbolt/errors"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	ErrArgs      = errors.New("no buckets were given as arguments")
	ErrEmptyDB   = errors.New("the database is empty with no buckets")
	ErrToFewArgs = errors.New("too few arguments were given")
	ErrCommand   = errors.New("command is unknown")
	ErrNilFlags  = errors.New("flags cannot be a nil value")
	ErrNoArgs    = errors.New("arguments cannot be empty")
	ErrUserExit  = errors.New("cannot dupe check a directory that isn't stored as a bucket")
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
		return bberr.ErrDatabaseNotOpen
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
		return move(db, c, assumeYes, args...)
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

func move(db *bolt.DB, c *dupe.Config, assumeYes bool, args ...string) error {
	const src, dest = 1, 2
	s, d := "", ""
	if len(args) > src {
		s = args[src]
	}
	if len(args) > dest {
		d = args[dest]
	}
	return bucket.Move(db, c, assumeYes, s, d)
}

// Dupe parses the dupe command.
func Dupe(db *bolt.DB, c *dupe.Config, f *cmd.Flags, args ...string) error {
	if db == nil {
		return bberr.ErrDatabaseNotOpen
	}
	if f == nil || f.Version == nil {
		return ErrNilFlags
	}
	c.Debugger("dupe command: " + strings.Join(args, " "))

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
		duplicate.Check(l, 0, minArgs)
		return ErrToFewArgs
	case l < minArgs:
		if len(b) == 0 {
			duplicate.Check(l, 0, minArgs)
			return ErrEmptyDB
		}
		return ErrArgs
	}
	if err := c.SetSource(args[source]); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			printer.StderrCR(os.ErrNotExist)
			fmt.Fprintf(os.Stdout, "File or directory path: %s\n", args[source])
			printer.Example("\ndupers dupe <file or directory> [buckets to lookup]")
		}
		return err
	}
	if err := SetStat(db, c, args...); err != nil {
		return err
	}
	return WalkScan(db, c, f, args...)
}

// SetStat sets and stats directories and files to scan, a bucket is the name given to database tables.
func SetStat(db *bolt.DB, c *dupe.Config, args ...string) error {
	if db == nil {
		return bberr.ErrDatabaseNotOpen
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
		c.Debugger("use all buckets: " + c.BucketS())
		return nil
	}
	if err := c.SetBuckets(buckets...); err != nil {
		var pathError *fs.PathError
		if errors.As(err, &pathError) {
			printer.StderrCR(bberr.ErrBucketNotFound)
			fmt.Fprintf(os.Stdout, "Bucket: %s\n", pathError.Path)
			printer.Example("\ndupers dupe " + args[1] + " [buckets to lookup]")
		}
		return err
	}
	if err := StatSource(c); err != nil {
		var pathError *fs.PathError
		if errors.As(err, &pathError) {
			printer.StderrCR(os.ErrNotExist)
			fmt.Fprintf(os.Stdout, "Bucket: %s\n", pathError.Path)
			printer.Example("\ndupers dupe <file or directory> [buckets to lookup]")
		}
		return err
	}
	c.Debugger("use buckets: " + c.BucketS())
	return nil
}

func WalkScan(db *bolt.DB, c *dupe.Config, f *cmd.Flags, args ...string) error {
	if db == nil {
		return bberr.ErrDatabaseNotOpen
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
	if err := duplicate.Cleanup(c, f); err != nil {
		return err
	}
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
		return bberr.ErrDatabaseNotOpen
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
	m, err := search.Compare(db, f, term, buckets)
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
		humanize.Bytes(safesize(writ)), name)
	printer.Quiet(quiet, s)
	return nil
}

func safesize(i int64) uint64 {
	if i < 0 {
		return 0
	}
	return uint64(i)
}

// cleanupDB cleans and compacts the database.
func CleanupDB(db *bolt.DB, c *dupe.Config) error {
	if db == nil {
		return bberr.ErrDatabaseNotOpen
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

// StatSource checks the path arguments supplied to the dupe command.
func StatSource(c *dupe.Config) error {
	if c == nil {
		return dupe.ErrNilConfig
	}
	isDir, files, verses, err := c.StatSource()
	if err != nil {
		return err
	}
	// handle any problems
	p := message.NewPrinter(language.English)
	verb := "Buckets"
	if len(c.Buckets) == 1 {
		verb = "Bucket"
	}
	src := c.GetSource()
	w := os.Stdout
	if isDir {
		fmt.Fprint(w, "Directory to check:")
	} else {
		fmt.Fprint(w, "File to check:")
	}
	fmt.Fprintln(w)
	if verses == 0 {
		fmt.Fprintf(w, " %s ", src)
	} else {
		fmt.Fprintf(w, " %s ", color.Warn.Sprint(src))
	}
	if !isDir {
		stat, err := os.Stat(src)
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "(%s)", color.Info.Sprintf("%s bytes", p.Sprint(stat.Size())))
	} else {
		fmt.Fprintf(w, "(%s)", color.Info.Sprintf("%s files", p.Sprint(files)))
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s to lookup, for finding duplicates:", verb)
	fmt.Fprintln(w)
	fmt.Fprintf(w, " %s ", c.BucketS())
	if verses == 0 {
		fmt.Fprintf(w, "(%s)", color.Danger.Sprintf("%s files", p.Sprint(verses)))
		fmt.Fprintln(w)
		fmt.Fprintln(w)
		fmt.Fprintln(os.Stderr, color.Danger.Sprintf("The %s to lookup contains no files", strings.ToLower(verb)))
		return bucket.ErrBucketEmpty
	}
	fmt.Fprintf(w, "(%s)", color.Info.Sprintf("%s files", p.Sprint(verses)))
	fmt.Fprintln(w)
	fmt.Fprintln(w)
	if isDir {
		return dirPrompt(w, c)
	}
	return nil
}

func dirPrompt(w io.Writer, c *dupe.Config) error {
	fmt.Fprintln(w, "About to scan and save this directory to the database:")
	fmt.Fprintf(w, " %s", c.BucketS())
	fmt.Fprintln(w)
	fmt.Fprintln(w)
	if !printer.AskYN("Is this what you want (no will exit)", c.Yes, printer.Yes) {
		return ErrUserExit
	}
	return nil
}
