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
// TODO drop c.Parser.DB
func Database(c *dupe.Config, assumeYes bool, args ...string) error {
	if c == nil {
		c = new(dupe.Config)
	}
	if _, err := database.Check(); err != nil {
		return err
	}
	if len(args) == 0 {
		return ErrCmd
	}
	buckets := [2]string{}
	copy(buckets[:], args)
	switch args[0] {
	case Backup_:
		fmt.Printf("%+v\n\n", c)
		return backupDB(c.Quiet)
	case Clean_:
		return cleanupDB(c)
	case DB_, Database_:
		s, err := database.Info()
		if err != nil {
			out.ErrCont(err)
		}
		fmt.Fprintln(os.Stdout, s)
	case Export_:
		bucket.Export(c.DB, c.Quiet, buckets)
	case Import_:
		bucket.Import(c.Parser.DB, c.Quiet, assumeYes, buckets)
	case LS_:
		return bucket.List(c.Parser.DB, c.Quiet, buckets)
	case MV_:
		buckets := [3]string{}
		copy(buckets[:], args)
		bucket.Move(c, assumeYes, buckets)
	case RM_:
		return bucket.Remove(c.Parser.DB, c.Quiet, assumeYes, buckets)
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
	b, err := database.All(c.Parser.DB)
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
	if err := walkCheck(c, *f.Yes, args...); err != nil {
		return err
	}
	return walkScan(c, f, args...)
}

// walkCheck checks directories and files to scan, a bucket is the name given to database tables.
func walkCheck(c *dupe.Config, assumeYes bool, args ...string) error {
	buckets := args[2:]
	if len(buckets) == 0 {
		if err := c.SetAllBuckets(c.DB); err != nil {
			return err
		}
		c.DPrint(fmt.Sprintf("use all buckets: %s", c.PrintBuckets()))
		return nil
	}
	if err := c.SetBuckets(buckets...); err != nil {
		return err
	}
	if code := checkDupePaths(c, assumeYes); code >= 0 {
		os.Exit(code)
	}
	c.DPrint(fmt.Sprintf("use buckets: %s", c.PrintBuckets()))
	return nil
}

func walkScan(c *dupe.Config, f *cmd.Flags, args ...string) error {
	// files or directories to compare (these are not saved to database)
	if err := c.WalkSource(); err != nil {
		return err
	}
	c.DPrint("walksource complete.")
	// walk, scan and save file paths and hashes to the database
	if err := duplicate.Lookup(c, f); err != nil {
		return err
	}
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
	name, writ, err := database.Backup()
	if err != nil {
		return err
	}
	s := fmt.Sprintf("A new copy of the database (%s) is at: %s",
		humanize.Bytes(uint64(writ)), name)
	out.Response(s, quiet)
	return nil
}

// cleanupDB cleans and compacts the database.
// c *dupe.Config
// db *bolt.DB, quiet, debug bool
func cleanupDB(c *dupe.Config) error {
	if err := database.Clean(c.Parser.DB, c.Quiet, c.Debug); err != nil {
		if b := errors.Is(err, database.ErrClean); !b {
			return err
		}
		out.ErrCont(err)
	}
	if err := database.Compact(c.Debug); err != nil {
		if b := errors.Is(err, database.ErrCompact); !b {
			return err
		}
	}
	return nil
}

// checkDupePaths checks the path arguments supplied to the dupe command.
// An os.exit code is return or a -1 for no errors.
func checkDupePaths(c *dupe.Config, assumeYes bool) (code int) {
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
	if buckets == 0 {
		fmt.Fprintf(w, " %s ", c.ToCheck())
	} else {
		fmt.Fprintf(w, " %s ", color.Warn.Sprint(c.ToCheck()))
	}
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
	fmt.Fprintln(w, "The bucket to lookup is to be stored in the database,")
	color.Warn.Println(" but the \"Directory to check\" is not.")
	if !out.YN("Is this what you want", assumeYes, out.No) {
		return 0
	}
	return -1
}
