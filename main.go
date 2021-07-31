// © Ben Garrett https://github.com/bengarrett/dupers

// Dupers is the blazing-fast file duplicate checker and filename search.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/bengarrett/dupers/database"
	"github.com/bengarrett/dupers/dupers"
	"github.com/bengarrett/dupers/out"
	"github.com/dustin/go-humanize"
	"github.com/gookit/color"
)

var (
	version = "0.0.0"
	commit  = "unset" // nolint: gochecknoglobals
	date    = "unset" // nolint: gochecknoglobals
)

var (
	ErrCmd    = errors.New("command is unknown")
	ErrNoArgs = errors.New("request is missing arguments")
	ErrNoDB   = errors.New("database has no bucket name")
	ErrSearch = errors.New("search request needs an expression")
)

const (
	winOS = "windows"
	dbf   = "database"
	dbs   = "db"
	dbk   = "backup"
	dcn   = "clean"
	dls   = "ls"
	drm   = "rm"
	dup   = "up"
	dupp  = "up+"
)

type tasks struct {
	debug    *bool
	exact    *bool
	filename *bool
	lookup   *bool
	quiet    *bool
	rm       *bool
	rmPlus   *bool
	sensen   *bool
}

func main() {
	var (
		c dupers.Config
		t tasks
	)
	c.SetTimer()
	// dupe options
	t.lookup = flag.Bool("fast", false, "query the database for a much faster match,"+
		"\n\t\tthe results maybe stale as it does not look for any file changes on your system")
	f := flag.Bool("f", false, "alias for fast")
	t.rm = flag.Bool("delete", false, "delete the duplicate files found in the <directory to check>")
	t.rmPlus = flag.Bool("delete+", false, "delete the duplicate files and remove empty directories from the <directory to check>")
	t.sensen = flag.Bool("sensen", false, "delete all files & directories other than unique .exe, .com programs in the <directory to check>")
	// search options
	t.exact = flag.Bool("exact", false, "match case")
	ex := flag.Bool("e", false, "alias for exact")
	t.filename = flag.Bool("name", false, "search for filenames, and ignore directories")
	fn := flag.Bool("n", false, "alias for name")
	// general options
	t.quiet = flag.Bool("quiet", false, "quiet mode hides all but essential feedback")
	q := flag.Bool("q", false, "alias for quiet")
	ver := flag.Bool("version", false, "version and information for this program")
	v := flag.Bool("v", false, "alias for version")
	// hidden flag
	t.debug = flag.Bool("debug", false, "debug mode")
	// help and parse flags
	flag.Usage = func() {
		help()
	}
	flag.Parse()
	if *q || *t.quiet {
		*t.quiet = true
		c.Quiet = true
	}
	if *t.debug {
		c.Debug = true
	}
	if s := options(ver, v); s != "" {
		fmt.Print(s)
		os.Exit(0)
	}

	selection := strings.ToLower(flag.Args()[0])
	if c.Debug {
		out.Bug("command selection: " + selection)
	}
	switch selection {
	case dbf, dbs, dbk, dcn, dls, drm, dup, dupp:
		taskDatabase(&c, *t.quiet, flag.Args()...)
	case "dupe":
		if *f {
			*t.lookup = true
		}
		taskScan(&c, t, flag.Args()...)
	case "search":
		if *ex {
			t.exact = ex
		}
		if *fn {
			t.filename = fn
		}
		taskSearch(t, flag.Args()...)
	default:
		out.ErrCont(ErrCmd)
		fmt.Printf("Command: '%s'\n\nSee the help for the available commands and options:\n", selection)
		out.Example("dupers --help")
		out.ErrFatal(nil)
	}
}

func taskDatabase(c *dupers.Config, quiet bool, args ...string) {
	var arr [2]string
	switch args[0] {
	case dbk:
		n, w, err := database.Backup()
		if err != nil {
			out.ErrFatal(err)
		}
		s := fmt.Sprintf("A new copy of the database (%s) is at: %s", humanize.Bytes(uint64(w)), n)
		out.Response(s, quiet)
		return
	case dcn:
		if err := database.Clean(quiet, c.Debug); err != nil {
			if b := errors.Is(err, database.ErrDBClean); !b {
				out.ErrFatal(err)
			}
			out.ErrCont(err)
		}
		if err := database.Compact(c.Debug); err != nil {
			if b := errors.Is(err, database.ErrDBCompact); !b {
				out.ErrFatal(err)
			}
		}
		return
	case dbs, dbf:
		s, err := database.Info()
		if err != nil {
			out.ErrCont(err)
		}
		fmt.Println(s)
		return
	case dls:
		copy(arr[:], args)
		taskDBList(quiet, arr)
	case drm:
		copy(arr[:], args)
		taskDBRM(quiet, arr)
		return
	case dup:
		copy(arr[:], args)
		taskDBUp(c, false, arr)
		return
	case dupp:
		copy(arr[:], args)
		taskDBUp(c, true, arr)
		return
	default:
		out.ErrFatal(ErrCmd)
	}
}

func taskDBList(quiet bool, args [2]string) {
	if args[1] == "" {
		out.ErrCont(ErrNoDB)
		fmt.Println("Cannot list the bucket as no bucket name was provided.")
		out.Example("\ndupers ls <bucket name>")
		out.ErrFatal(nil)
	}
	name, err := filepath.Abs(args[1])
	if err != nil {
		out.ErrFatal(err)
	}
	ls, err := database.List(name)
	if err != nil {
		out.ErrCont(err)
	}
	// sort the filenames
	var names []string
	for name := range ls {
		names = append(names, string(name))
	}
	sort.Strings(names)
	for _, name := range names {
		sum := ls[database.Filepath(name)]
		fmt.Printf("%x %s\n", sum, name)
	}
	if cnt := len(ls); !quiet && cnt > 0 {
		fmt.Printf("%s %s\n", color.Primary.Sprint(cnt),
			color.Secondary.Sprint("items listed. Checksums are 32 byte, SHA-256 (FIPS 180-4)."))
	}
}

func taskDBRM(quiet bool, args [2]string) {
	if args[1] == "" {
		out.ErrCont(ErrNoDB)
		fmt.Println("Cannot remove a bucket from the database as no bucket name was provided.")
		out.Example("\ndupers rm <bucket name>")
		out.ErrFatal(nil)
	}
	name, err := filepath.Abs(args[1])
	if err != nil {
		out.ErrFatal(err)
	}
	if err := database.RM(name); err != nil {
		if errors.Is(err, database.ErrNoBucket) {
			// retry with the original argument
			if err1 := database.RM(args[1]); err1 != nil {
				if errors.Is(err1, database.ErrNoBucket) {
					out.ErrCont(err1)
					fmt.Printf("Bucket to remove: %s\n", color.Danger.Sprint(name))
					buckets, err2 := database.AllBuckets()
					if err2 != nil {
						out.ErrFatal(err2)
					}
					fmt.Printf("Buckets in use:   %s\n", strings.Join(buckets, "\n\t\t  "))
					out.ErrFatal(nil)
				}
				out.ErrFatal(err1)
			}
		}
	}
	s := fmt.Sprintf("Removed bucket from the database: '%s'\n", name)
	out.Response(s, quiet)
}

func taskDBUp(c *dupers.Config, plus bool, args [2]string) {
	if args[1] == "" {
		out.ErrCont(database.ErrNoBucket)
		fmt.Println("Cannot add or update a bucket to the database as no bucket name was provided.")
		if plus {
			out.Example("\ndupers up+ <bucket name>")
			out.ErrFatal(nil)
		}
		out.Example("\ndupers up <bucket name>")
		out.ErrFatal(nil)
	}
	path, err := filepath.Abs(args[1])
	if err != nil {
		out.ErrFatal(err)
	}
	name := dupers.Bucket(path)
	if runtime.GOOS == winOS && !c.Quiet {
		fmt.Printf("To improve performance on Windows use the quiet flag: %s\n",
			color.Debug.Sprintf("duper -quiet up '%s'", path))
	}
	if plus {
		if err := c.WalkArchiver(name); err != nil {
			out.ErrFatal(err)
		}
	} else if err := c.WalkDir(name); err != nil {
		out.ErrFatal(err)
	}
	if runtime.GOOS == winOS || !c.Quiet {
		fmt.Println(c.Status())
	}
}

func taskScan(c *dupers.Config, t tasks, args ...string) {
	if c.Debug {
		s := fmt.Sprintf("taskScan: %s", strings.Join(args, " "))
		out.Bug(s)
	}
	l := len(args)
	b, err := database.AllBuckets()
	if err != nil {
		out.ErrFatal(err)
	}
	const minArgs = 3
	if l < minArgs && len(b) == 0 {
		taskScanErr(l, len(b))
	}
	// directory or a file to match
	c.SetToCheck(args[1])
	// directories and files to scan, a bucket is the name given to database tables
	arr := args[2:]
	c.SetBuckets(arr...)
	if arr == nil {
		c.SetAllBuckets()
	}
	if c.Debug {
		s := fmt.Sprintf("buckets: %s", c.PrintBuckets())
		out.Bug(s)
	}
	taskCheckPaths(c)
	// files or directories to compare (these are not saved to database)
	if err := c.WalkSource(); err != nil {
		out.ErrFatal(err)
	}
	if c.Debug {
		out.Bug("walksource complete.")
	}
	// windows notice
	if runtime.GOOS == winOS && !*t.quiet {
		fmt.Printf("To improve performance on Windows use the quiet flag: %s\n",
			color.Debug.Sprintf("duper -quiet dupe '%s' '%s'", c.ToCheck(), c.PrintBuckets()))
	}
	// walk, scan and save file paths and hashes to the database
	taskLookup(c, t)
	// print the found dupes & remove files
	taskScanClean(c, t)
	// summaries
	if runtime.GOOS == winOS || !c.Quiet {
		fmt.Println(c.Status())
	}
}

func taskLookup(c *dupers.Config, t tasks) {
	if !*t.lookup {
		if c.Debug {
			out.Bug("database cleanup.")
		}
		if err := database.Clean(true, c.Debug); err != nil {
			out.ErrCont(err)
		}
		if c.Debug {
			out.Bug("walk the buckets.")
		}
		c.WalkDirs()
	} else {
		if c.Debug {
			out.Bug("seek in buckets.")
		}
		fmt.Print(c.Seek())
	}
}

func taskScanClean(c *dupers.Config, t tasks) {
	fmt.Print(c.Print())
	if *t.rm || *t.rmPlus {
		if c.Debug {
			out.Bug("remove duplicate files.")
		}
		fmt.Print(c.Remove())
	}
	if *t.sensen {
		if c.Debug {
			out.Bug("remove all non unique Windows and MS-DOS files.")
		}
		fmt.Print(c.RemoveAll(*t.rmPlus))
		fmt.Print(c.Remove())
	} else if *t.rmPlus {
		if c.Debug {
			out.Bug("remove empty directories.")
		}
		fmt.Print(c.Clean())
	}
}

func taskSearch(t tasks, args ...string) {
	l := len(args)
	taskExpErr(l)
	term := args[1]
	var (
		buckets = []string{}
		m       *database.Matches
		err     error
	)
	const minArgs = 2
	if l > minArgs {
		buckets = args[2:]
	}
	if *t.filename {
		if !*t.exact {
			if m, err = database.CompareBaseNoCase(term, buckets...); err != nil {
				taskSearchErr(err)
			}
		}
		if *t.exact {
			if m, err = database.CompareBase(term, buckets...); err != nil {
				taskSearchErr(err)
			}
		}
	}
	if !*t.filename {
		if !*t.exact {
			if m, err = database.CompareNoCase(term, buckets...); err != nil {
				taskSearchErr(err)
			}
		}
		if *t.exact {
			if m, err = database.Compare(term, buckets...); err != nil {
				taskSearchErr(err)
			}
		}
	}
	fmt.Print(dupers.Print(*t.quiet, m))
	if !*t.quiet {
		l := 0
		if m != nil {
			l = len(*m)
		}
		fmt.Println(searchSummary(l, term, *t.exact, *t.filename))
	}
}

func searchSummary(total int, term string, exact, filename bool) string {
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

func options(ver, v *bool) string {
	// convenience for when a help or version flag is passed as an argument
	for _, arg := range flag.Args() {
		switch strings.ToLower(arg) {
		case "-h", "-help", "--help":
			return help()
		case "-v", "-version", "--version":
			return info()
		}
	}
	// print version information
	if *ver || *v {
		return info()
	}
	// print help if no arguments are given
	if len(flag.Args()) == 0 {
		return help()
	}
	return ""
}
