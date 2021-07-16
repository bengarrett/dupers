// © Ben Garrett https://github.com/bengarrett/dupers

// dupers todo
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"
	"time"

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
	drm   = "rm"
	dup   = "up"
)

// --deep-scan (open archives and hash binary files)

func main() {
	var c dupers.Config
	c.Timer = time.Now()
	alert := "\n\t\tthere are no prompts, " + color.Danger.Sprint("use this flag at your own risk")
	// dupe options
	look := flag.Bool("fast", false, "query the database for a much faster match,"+
		"\n\t\tthe results maybe stale as it does not look for any file changes on your system")
	f := flag.Bool("f", false, "alias for fast")
	rmdupe := flag.Bool("delete", false, "delete the duplicate files found in the <directory or file to check>")
	sensen := flag.Bool("sensen", false, "purges everything other than unique .exe and .com programs in the <directory to check>"+alert)
	// search options
	exact := flag.Bool("exact", false, "match case")
	ex := flag.Bool("e", false, "alias for exact")
	filename := flag.Bool("name", false, "search for filenames, and ignore directories")
	fn := flag.Bool("n", false, "alias for name")
	// general options
	quiet := flag.Bool("quiet", false, "quiet mode hides all but essential feedback")
	q := flag.Bool("q", false, "alias for quiet")
	ver := flag.Bool("version", false, "version and information for this program")
	v := flag.Bool("v", false, "alias for version")
	// help and parse flags
	flag.Usage = func() {
		help()
	}
	flag.Parse()
	if *q || *quiet {
		*quiet = true
		c.Quiet = true
	}
	options(ver, v)

	selection := strings.ToLower(flag.Args()[0])
	switch selection {
	case dbf, dbs, dbk, dcn, drm, dup:
		taskDatabase(&c, *quiet, flag.Args()...)
	case "dupe":
		if *f {
			*look = true
		}
		taskScan(&c, *look, *quiet, *rmdupe, *sensen, flag.Args()...)
	case "search":
		if *ex {
			exact = ex
		}
		if *fn {
			filename = fn
		}
		taskSearch(*exact, *filename, *quiet, flag.Args()...)
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
	f = flag.Lookup("sensen")
	fmt.Fprintf(w, "\n        -%v\t\t%v\n", f.Name, f.Usage)
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
	fmt.Fprintf(w, "    dupers %s <bucket>\t%s\n", drm, "remove the bucket (a scanned directory) from the database")
	fmt.Fprintf(w, "    dupers %s <bucket>\t%s\n", dup, "add or update the bucket (a directory to scan) to the database")
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
	fmt.Fprint(w, color.Info.Sprintf("    dupers dupe '%s' '%s'",
		filepath.Join(home(), "file.zip"), filepath.Join(home(), "Downloads")))
	fmt.Fprint(w, color.Note.Sprint("\t# find identical copies of file.zip in the Downloads directory\n"))
	fmt.Fprint(w, color.Info.Sprintf("    dupers -fast dupe '%s' '%s'",
		"doc.txt", filepath.Join(home(), "Documents")))
	fmt.Fprint(w, color.Note.Sprint("\t\t# use the database to find doc.txt in the Documents directory\n"))
	if runtime.GOOS == winOS {
		fmt.Fprint(w, color.Info.Sprintf("    dupers dupe '%s' %s %s",
			filepath.Join(home(), "Documents"), "D:", "E:"))
		fmt.Fprint(w, color.Note.Sprint("\t\t# search for files in Documents that also exist on drives D: and E:\n"))
	} else {
		fmt.Fprint(w, color.Info.Sprintf("    dupers dupe '%s' '%s'",
			filepath.Join(home(), "Documents"), "/var/www"))
		fmt.Fprint(w, color.Note.Sprint("\t\t# search for files in Documents that also exist in /var/www\n"))
	}
	return w
}

func exampleSearch(w *tabwriter.Writer) *tabwriter.Writer {
	fmt.Fprintln(w, "\n  Examples:")
	fmt.Fprint(w, "    "+color.Info.Sprintf("dupers search 'foo' '%s'", home()))
	fmt.Fprint(w, color.Note.Sprint("\t# search for the expression foo in your home directory\n"))
	fmt.Fprint(w, "    "+color.Info.Sprint("dupers search 'bar'"))
	fmt.Fprint(w, color.Note.Sprint("\t\t# search for the expression bar in the database\n"))
	fmt.Fprint(w, "    "+color.Info.Sprint("dupers -name search '.zip'"))
	fmt.Fprint(w, color.Note.Sprint("\t\t# search for filenames containing .zip\n"))
	return w
}

func taskDatabase(c *dupers.Config, quiet bool, args ...string) {
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
		if err := database.Clean(quiet); err != nil {
			if b := errors.Is(err, database.ErrDBClean); !b {
				out.ErrFatal(err)
			}
			out.ErrCont(err)
		}
		if err := database.Compact(); err != nil {
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
	case drm:
		taskDBRM(quiet, args...)
		return
	case dup:
		taskDBUp(c, args...)
		return
	default:
		out.ErrFatal(ErrCmd)
	}
}

func taskDBRM(quiet bool, args ...string) {
	const minArgs = 1
	l := len(args)
	if l == minArgs {
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
			out.ErrCont(err)
			fmt.Printf("Bucket to remove: '%s'\n", name)
			buckets, err1 := database.Buckets()
			if err1 != nil {
				out.ErrFatal(err1)
			}
			fmt.Printf("Buckets in use: %s\n", strings.Join(buckets, "\n\t\t"))
			out.ErrFatal(nil)
		}
		out.ErrFatal(err)
	}
	s := fmt.Sprintf("Removed bucket from the database: '%s'\n", name)
	out.Response(s, quiet)
}

func taskDBUp(c *dupers.Config, args ...string) {
	const minArgs = 1
	l := len(args)
	if l == minArgs {
		out.ErrCont(database.ErrNoBucket)
		fmt.Println("Cannot add or update a bucket to the database as no bucket name was provided.")
		out.Example("\ndupers database up <bucket name>")
		out.ErrFatal(nil)
	}
	path, err := filepath.Abs(args[1])
	if err != nil {
		out.ErrFatal(err)
	}
	if runtime.GOOS == winOS && !c.Quiet {
		fmt.Printf("To improve performance on Windows use the quiet flag: duper -quiet dupe %s %s\n", c.Source, strings.Join(c.Buckets, " "))
	}
	if err := c.WalkDir(path); err != nil {
		out.ErrFatal(err)
	}
	if runtime.GOOS == winOS || !c.Quiet {
		fmt.Println(c.Status())
	}
}

func taskScan(c *dupers.Config, lookup, quiet, rm, sensen bool, args ...string) {
	l := len(args)
	b, err := database.Buckets()
	if err != nil {
		out.ErrFatal(err)
	}
	const minArgs = 3
	if l < minArgs && len(b) == 0 {
		taskScanErr(l, len(b))
	}
	// directory or a file to match
	c.Source = args[1]
	// directories and files to scan, a bucket is the name given to database tables
	c.Buckets = args[2:]
	if l < minArgs {
		c.Buckets = b
	}
	// files or directories to compare (these are not saved to database)
	c.WalkSource()
	// windows notice
	if runtime.GOOS == winOS && !quiet {
		fmt.Printf("To improve performance on Windows use the quiet flag: duper -quiet dupe %s %s\n", c.Source, strings.Join(c.Buckets, " "))
	}
	// walk, scan and save file paths and hashes to the database
	if !lookup {
		if err := database.Clean(true); err != nil {
			out.ErrCont(err)
		}
		c.WalkDirs()
	} else {
		c.Seek()
	}
	// print the found dupes
	c.Print()
	// remove files
	if rm || sensen {
		c.Remove()
		if sensen {
			c.PurgeSrc()
		}
	}
	// summaries
	if runtime.GOOS == winOS || !c.Quiet {
		fmt.Println(c.Status())
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

func taskSearch(exact, filename, quiet bool, args ...string) {
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
	if filename {
		if !exact {
			if m, err = database.CompareBaseNoCase(term, buckets); err != nil {
				taskSearchErr(err)
			}
		}
		if exact {
			if m, err = database.CompareBase(term, buckets); err != nil {
				taskSearchErr(err)
			}
		}
	}
	if !filename {
		if !exact {
			if m, err = database.CompareNoCase(term, buckets); err != nil {
				taskSearchErr(err)
			}
		}
		if exact {
			if m, err = database.Compare(term, buckets); err != nil {
				taskSearchErr(err)
			}
		}
	}
	dupers.Print(term, quiet, m)
	if !quiet {
		l := 0
		if m != nil {
			l = len(*m)
		}
		fmt.Println(searchSummary(l, term, exact, filename))
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
