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
	"github.com/bengarrett/dupers/internal/task/internal/help"
	"github.com/bengarrett/dupers/internal/task/internal/search"
	"github.com/dustin/go-humanize"
	"github.com/gookit/color"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	ErrArgs = errors.New("no buckets were given as arguments")
	ErrCmd  = errors.New("command is unknown")
	ErrNil  = errors.New("argument cannot be a nil value")
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
	if err := database.Check(); err != nil {
		return err
	}
	buckets := [2]string{}
	copy(buckets[:], args)
	switch args[0] {
	case dbk:
		return backupDB(quiet)
	case dcn:
		return cleanupDB(quiet, c.Debug)
	case dbs, dbf:
		s, err := database.Info()
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
func Dupe(c *dupe.Config, f *cmd.Flags, args ...string) error {
	if c == nil || f == nil {
		return ErrNil
	}
	if c.Debug {
		s := fmt.Sprintf("dupeCmd: %s", strings.Join(args, " "))
		out.PBug(s)
	}
	l := len(args)
	if l == 1 {
		const minArgs = 2
		duplicate.CmdErr(l, 0, minArgs, false)
	}
	// fetch bucket info
	b, err := database.All(nil)
	if err != nil {
		return err
	}
	const minArgs = 3
	if l < minArgs && len(b) == 0 {
		duplicate.CmdErr(l, len(b), minArgs, false)
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
			out.ErrFatal(err)
		}
	} else {
		c.SetBucket(buckets...)
		checkDupePaths(c)
	}
	if c.Debug {
		s := fmt.Sprintf("buckets: %s", c.PrintBuckets())
		out.PBug(s)
	}
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

// Search parses the commands that handle search.
func Search(f *cmd.Flags, args ...string) {
	l := len(args)
	search.CmdErr(l, false)
	term, buckets := args[1], []string{}
	const minArgs = 2
	if l > minArgs {
		buckets = args[minArgs:]
	}
	m, err := search.Compare(f, term, buckets, false)
	if err != nil {
		out.ErrFatal(err)
	}
	fmt.Print(dupe.Print(*f.Quiet, *f.Exact, term, m))
	if !*f.Quiet {
		l := 0
		if m != nil {
			l = len(*m)
		}
		fmt.Println(cmd.SearchSummary(l, term, *f.Exact, *f.Filename))
	}
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
