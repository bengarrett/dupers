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
	"text/tabwriter"

	dupers "github.com/bengarrett/dupers/lib"
	"github.com/bengarrett/dupers/lib/database"
	"github.com/bengarrett/dupers/lib/out"
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
	options(ver, v)

	selection := strings.ToLower(flag.Args()[0])
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

// Help, usage and examples.
func help() {
	var f *flag.Flag
	w := tabwriter.NewWriter(os.Stderr, 0, 0, 4, ' ', 0)
	fmt.Fprintf(w, "Dupers is the blazing-fast file duplicate checker and filename search.\n")
	windowsNotice(w)
	fmt.Fprintf(w, "\n%s\n  Scan for duplicate files, matching files that share the identical content.\n",
		color.Primary.Sprint("Dupe:"))
	fmt.Fprintln(w, "  The \"directory or file to check\" is never added to the database.")
	if runtime.GOOS == winOS {
		fmt.Fprintln(w, "  The \"buckets to lookup\" are directories or drive letters that get added to the database for quicker scans.")
	} else {
		fmt.Fprintln(w, "  The \"buckets to lookup\" are directories that get added to the database for quicker scans.")
	}
	fmt.Fprintln(w, "\n  Usage:")
	fmt.Fprintln(w, "    dupers [options] dupe <directory or file to check> [buckets to lookup]")
	fmt.Fprintln(w, "\n  Options:")
	f = flag.Lookup("fast")
	fmt.Fprintf(w, "    -%v, -%v\t\t%v\n", f.Name[:1], f.Name, f.Usage)
	f = flag.Lookup("delete")
	fmt.Fprintf(w, "        -%v\t\t%v\n", f.Name, f.Usage)
	f = flag.Lookup("delete+")
	fmt.Fprintf(w, "        -%v\t\t%v\n", f.Name, f.Usage)
	f = flag.Lookup("sensen")
	fmt.Fprintf(w, "        -%v\t\t%v\n", f.Name, f.Usage)
	exampleDupe(w)
	fmt.Fprintf(w, "\n%s\n  Lookup a file or a directory name in the database.\n",
		color.Primary.Sprint("Search:"))
	fmt.Fprintf(w, "  The <search expression> can be a partial or complete, file or directory name.\n")
	fmt.Fprintln(w, "\n  Usage:")
	fmt.Fprintln(w, "    dupers [options] search <search expression> [optional, buckets to search]")
	fmt.Fprintln(w, "\n  Options:")
	f = flag.Lookup("exact")
	fmt.Fprintf(w, "    -%v, -%v\t\t%v\n", f.Name[:1], f.Name, f.Usage)
	f = flag.Lookup("name")
	fmt.Fprintf(w, "    -%v, -%v\t\t%v\n", f.Name[:1], f.Name, f.Usage)
	exampleSearch(w)

	fmt.Fprintf(w, "\n%s\n  View information and run optional maintenance on the internal database.\n",
		color.Primary.Sprint("Database:"))
	fmt.Fprintln(w, "\n  Usage:")
	fmt.Fprintf(w, "    dupers %s\tdisplay statistics and bucket information\n", dbf)
	fmt.Fprintf(w, "    dupers %s\t%s\n", dbk, "make a copy of the database to: "+home())
	fmt.Fprintf(w, "    dupers %s\t%s\n", dcn, "compact and remove all items in the database that point to missing files")
	fmt.Fprintf(w, "    dupers %s  <bucket>\t%s\n", dls, "list the hashes and files in the bucket")
	fmt.Fprintf(w, "    dupers %s  <bucket>\t%s\n", drm, "remove the bucket from the database")
	fmt.Fprintf(w, "    dupers %s  <bucket>\t%s\n", dup, "add or update the bucket to the database")
	fmt.Fprintf(w, "    dupers %s <bucket>\t%s\n", dupp, "add or update the bucket using an archive scan "+
		color.Danger.Sprint("(SLOW)")+
		"\n\tthe scan reads all the files stored within file archives")

	fmt.Fprintln(w, "\nOptions:")
	f = flag.Lookup("quiet")
	fmt.Fprintf(w, "    -%v, -%v\t%v\n", f.Name[:1], f.Name, f.Usage)
	f = flag.Lookup("version")
	fmt.Fprintf(w, "    -%v, -%v\t%v\n", f.Name[:1], f.Name, f.Usage)
	fmt.Fprintln(w, "    -h, -help\tshow this list of options")
	fmt.Fprintln(w)
	w.Flush()
}

func windowsNotice(w *tabwriter.Writer) *tabwriter.Writer {
	if runtime.GOOS != winOS {
		return w
	}
	empty, err := database.IsEmpty()
	if err != nil {
		out.ErrCont(err)
	}
	if empty {
		fmt.Fprintf(w, "\n%s\n", color.Danger.Sprint("To greatly improve performance,"+
			" please apply Windows Security Exclusions to the directories to be scanned."))
	}
	return w
}

func exampleDupe(w *tabwriter.Writer) *tabwriter.Writer {
	fmt.Fprintln(w, "\n  Examples:")
	fmt.Fprint(w, color.Secondary.Sprint("    # find identical copies of file.zip in the Downloads directory\n"))
	fmt.Fprint(w, color.Info.Sprintf("    dupers dupe '%s' '%s'",
		filepath.Join(home(), "file.zip"), filepath.Join(home(), "Downloads")))

	if runtime.GOOS == winOS {
		fmt.Fprint(w, color.Secondary.Sprint("\n    # search for files in Documents that also exist on drives D: and E:\n"))
		fmt.Fprint(w, color.Info.Sprintf("    dupers dupe '%s' %s %s",
			filepath.Join(home(), "Documents"), "D:", "E:"))
	} else {
		fmt.Fprint(w, color.Secondary.Sprint("\n    # search for files in Documents that also exist in /var/www\n"))
		fmt.Fprint(w, color.Info.Sprintf("    dupers dupe '%s' '%s'",
			filepath.Join(home(), "Documents"), "/var/www"))
	}
	fmt.Fprintln(w)
	return w
}

func exampleSearch(w *tabwriter.Writer) *tabwriter.Writer {
	fmt.Fprintln(w, "\n  Examples:")
	fmt.Fprint(w, color.Secondary.Sprint("    # search for the expression foo in your home directory\n"))
	fmt.Fprint(w, "    "+color.Info.Sprintf("dupers search 'foo' '%s'", home()))
	fmt.Fprint(w, color.Secondary.Sprint("\n    # search for filenames containing .zip\n"))
	fmt.Fprint(w, "    "+color.Info.Sprint("dupers -name search '.zip'"))
	fmt.Fprintln(w)
	return w
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
		out.Example("\ndupers database ls <bucket name>")
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
		fmt.Printf("%d items listed. Checksums are 32 byte, SHA-256 (FIPS 180-4).\n", cnt)
	}
}

func taskDBRM(quiet bool, args [2]string) {
	if args[1] == "" {
		out.ErrCont(ErrNoDB)
		fmt.Println("Cannot remove a bucket from the database as no bucket name was provided.")
		out.Example("\ndupers database rm <bucket name>")
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
					fmt.Printf("Bucket to remove: '%s'\n", name)
					buckets, err2 := database.AllBuckets()
					if err2 != nil {
						out.ErrFatal(err2)
					}
					fmt.Printf("Buckets in use: %s\n", strings.Join(buckets, "\n\t\t"))
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
			out.Example("\ndupers database up+ <bucket name>")
			out.ErrFatal(nil)
		}
		out.Example("\ndupers database up <bucket name>")
		out.ErrFatal(nil)
	}
	path, err := filepath.Abs(args[1])
	if err != nil {
		out.ErrFatal(err)
	}
	if runtime.GOOS == winOS && !c.Quiet {
		fmt.Printf("To improve performance on Windows use the quiet flag: duper -quiet dupe %s %s\n", c.ToCheck(), strings.Join(c.Buckets, " "))
	}
	if plus {
		if err := c.WalkArchiver(path); err != nil {
			out.ErrFatal(err)
		}
	} else if err := c.WalkDir(path); err != nil {
		out.ErrFatal(err)
	}
	if runtime.GOOS == winOS || !c.Quiet {
		fmt.Println(c.Status())
	}
}

func taskScan(c *dupers.Config, t tasks, args ...string) {
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
	c.Buckets = args[2:]
	if l < minArgs {
		c.Buckets = b
	}
	// files or directories to compare (these are not saved to database)
	c.WalkSource()
	if c.Debug {
		out.Bug("walksource complete.")
	}
	// windows notice
	if runtime.GOOS == winOS && !*t.quiet {
		fmt.Printf("To improve performance on Windows use the quiet flag: duper -quiet dupe %s %s\n", c.ToCheck(), strings.Join(c.Buckets, " "))
	}
	// walk, scan and save file paths and hashes to the database
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
		c.Seek()
	}
	// print the found dupes & remove files
	taskScanClean(c, t)
	// summaries
	if runtime.GOOS == winOS || !c.Quiet {
		fmt.Println(c.Status())
	}
}

func taskScanClean(c *dupers.Config, t tasks) {
	if c.Debug {
		out.Bug("print duplicate results.")
	}
	c.Print()
	if *t.rm || *t.sensen {
		if c.Debug {
			out.Bug("remove duplicate files.")
		}
		c.Remove()
	}
	if *t.sensen {
		if c.Debug {
			out.Bug("remove all non unique Windows and MS-DOS files.")
		}
		c.RemoveAll(*t.rmPlus)
	} else if *t.rmPlus {
		if c.Debug {
			out.Bug("remove empty directories.")
		}
		c.Clean()
	}
}

func taskScanErr(args, buckets int) {
	const minArgs = 2
	if args < minArgs {
		out.ErrCont(ErrNoArgs)
		fmt.Println("\nThe dupe command requires both a source and target.")
		fmt.Println("The source can be either a directory or file.")
		if runtime.GOOS == winOS {
			fmt.Println("The target can be one or more directories or drive letters.")
		} else {
			fmt.Println("The target can be one or more directories.")
		}
		out.Example("\ndupers dupe <source file or directory> <target one or more directories>")
	}
	if buckets == 0 && args == minArgs {
		if runtime.GOOS == winOS {
			color.Warn.Println("the dupe request requires at least one target directory or drive letter")
		} else {
			color.Warn.Println("the dupe request requires at least one target directory")
		}
		s := fmt.Sprintf("\ndupers dupe %s <target one or more directories>\n", flag.Args()[1])
		out.Example(s)
	}
	out.ErrFatal(nil)
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
	dupers.Print(term, *t.quiet, m)
	if !*t.quiet {
		l := 0
		if m != nil {
			l = len(*m)
		}
		fmt.Println(searchSummary(l, term, *t.exact, *t.filename))
	}
}

func taskExpErr(l int) {
	if l <= 1 {
		out.ErrCont(ErrSearch)
		fmt.Println("A search expression can be a partial or complete filename,")
		fmt.Println("or a partial or complete directory.")
		out.Example("\ndupers search <search expression> [optional, directories to search]")
		out.ErrFatal(nil)
	}
}

func taskSearchErr(err error) {
	if errors.As(err, &database.ErrNoBucket) {
		out.ErrCont(err)
		fmt.Println("\nTo add this directory to the database, run:")
		dir := strings.ReplaceAll(err.Error(), errors.Unwrap(err).Error()+": ", "")
		s := fmt.Sprintf("dupers up %s\n", dir)
		out.Example(s)
		out.ErrFatal(nil)
	}
	out.ErrFatal(err)
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

func options(ver, v *bool) {
	// convenience for when a help or version flag is passed as an argument
	for _, arg := range flag.Args() {
		switch strings.ToLower(arg) {
		case "-h", "-help", "--help":
			help()
			os.Exit(0)
		case "-v", "-version", "--version":
			info()
			os.Exit(0)
		}
	}
	// print version information
	if *ver || *v {
		info()
		os.Exit(0)
	}
	// print help if no arguments are given
	if len(flag.Args()) == 0 {
		help()
		os.Exit(0)
	}
}

// Info prints out the program information and version.
func info() {
	const copyright = "\u00A9"
	fmt.Printf("dupers v%s\n%s 2021 Ben Garrett\n", version, copyright)
	fmt.Printf("https://github.com/bengarrett/dupers\n\n")
	fmt.Printf("build: %s (%s)\n", commit, date)
	exe, err := self()
	if err != nil {
		out.ErrFatal(err)
	}
	fmt.Printf("path: %s\n", exe)
}

func home() string {
	h, err := os.UserHomeDir()
	if err != nil {
		if h, err = os.Getwd(); err != nil {
			out.ErrCont(err)
		}
	}
	return h
}

func self() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("self error: %w", err)
	}
	return exe, nil
}
