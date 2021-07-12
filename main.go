// © Ben Garrett https://github.com/bengarrett/dupers

// dupers todo
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"
	"time"

	dupers "github.com/bengarrett/dupers/lib"
	"github.com/bengarrett/dupers/lib/database"
	"github.com/dustin/go-humanize"
	"github.com/gookit/color"
)

var (
	version = "0.0.0"
	commit  = "unset" // nolint: gochecknoglobals
	date    = "unset" // nolint: gochecknoglobals
)

const winOS = "windows"

// --delete-dupes
// --move-dupes
// --copy-dupes
// --deep-scan (open archives and hash binary files)

func main() {
	var c dupers.Config
	c.Timer = time.Now()

	look := flag.Bool("fast", false, "query the database for a much faster match,"+
		"\n\t\tthe results maybe stale as it does not look for any file changes on your system")
	f := flag.Bool("f", false, "alias for fast")
	exact := flag.Bool("exact", false, "match case")
	ex := flag.Bool("e", false, "alias for exact")
	filename := flag.Bool("name", false, "search for filenames, and ignore directories")
	fn := flag.Bool("n", false, "alias for name")

	quiet := flag.Bool("quiet", false, "quiet mode hides all but essential feedback")
	q := flag.Bool("q", false, "alias for quiet")
	ver := flag.Bool("version", false, "version and information for this program")
	v := flag.Bool("v", false, "alias for version")

	flag.Usage = func() {
		help()
	}
	flag.Parse()
	if *q {
		*quiet = true
	}
	options(ver, v)

	selection := strings.ToLower(flag.Args()[0])
	switch selection {
	case "database", "db", "backup", "clean", "rm", "up":
		taskDatabase(c, *quiet, flag.Args()...)
	case "dupe":
		if *f {
			*look = true
		}
		if *q || *quiet {
			c.Quiet = true
		}
		taskScan(&c, *look, *quiet)
	case "search":
		if *ex {
			exact = ex
		}
		if *fn {
			filename = fn
		}
		taskSearch(exact, filename, quiet)
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
	fmt.Fprintln(w, "    dupers [options] dupe <directory or file to check> <buckets to lookup>")
	fmt.Fprintln(w, "\n  Options:")
	f = flag.Lookup("fast")
	fmt.Fprintf(w, "    -%v, -%v\t\t%v\n", f.Name[:1], f.Name, f.Usage)
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

	fmt.Fprintf(w, "\n%s\n  Lookup some text or a word contained within a textfile.\n",
		color.Primary.Sprint("Text:"))
	fmt.Fprintln(w, "\n  Usage:")
	fmt.Fprintln(w, "    dupers [options] text <search text or word>")

	fmt.Fprintf(w, "\n%s\n  View information and run optional maintenance on the internal database.\n",
		color.Primary.Sprint("Database:"))
	fmt.Fprintln(w, "\n  Usage:")
	fmt.Fprintln(w, "    dupers database\tdisplay statistics and bucket information")
	fmt.Fprintf(w, "    dupers %s\t%s\n", "backup", "make a copy of the database to: "+home())
	fmt.Fprintf(w, "    dupers %s\t%s\n", "clean", "compact and remove all items in the database that point to missing files")
	fmt.Fprintf(w, "    dupers %s <bucket>\t%s\n", "rm", "remove the bucket (a scanned directory) from the database")
	fmt.Fprintf(w, "    dupers %s <bucket>\t%s\n", "up", "add or update the bucket (a directory to scan) to the database")
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
		log.Println(err)
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

func taskDatabase(c dupers.Config, quiet bool, args ...string) {
	l := len(args)
	const minArgs = 1
	switch args[0] {
	case "backup":
		n, w, err := database.Backup()
		if err != nil {
			log.Fatalln(err)
		}
		if !quiet {
			fmt.Printf("A new copy of the database (%s) is at: %s\n", humanize.Bytes(uint64(w)), n)
		}
		return
	case "clean":
		if err := database.Clean(quiet); err != nil {
			if b := errors.Is(err, database.ErrDBClean); !b {
				log.Fatalln(err)
			}
			fmt.Printf("The %s\n", err.Error())
		}
		if err := database.Compact(); err != nil {
			if b := errors.Is(err, database.ErrDBCompact); !b {
				log.Fatalln(err)
			}
		}
		return
	case "db", "database":
		fmt.Println(database.Info())
		return
	case "rm":
		if l == minArgs {
			color.Warn.Println("Cannot remove a bucket from the database as no bucket name was provided")
			fmt.Println("\ndupers database rm <bucket name>")
			fmt.Println()
			os.Exit(1)
		}
		name := args[1]
		if err := database.RM(name); err != nil {
			if errors.Is(err, database.ErrNoBucket) {
				fmt.Printf("The bucket does not exist in the database: '%s'\n", name)
				buckets, err1 := database.Buckets()
				if err1 != nil {
					log.Fatalln(err1)
				}
				fmt.Printf("Buckets in use: %s\n", strings.Join(buckets, "\n\t\t"))
				os.Exit(1)
			}
			log.Fatalln(err)
		}
		fmt.Printf("Removed bucket from the database: '%s'\n", name)
		return
	case "up":
		if l == minArgs {
			color.Warn.Println("Cannot add or update a bucket to the database as no bucket name was provided")
			fmt.Println("\ndupers database up <bucket name>")
			fmt.Println()
			os.Exit(1)
		}
		if runtime.GOOS == winOS && !quiet {
			fmt.Printf("To improve performance on Windows use the quiet flag: duper -quiet dupe %s %s\n", c.Source, strings.Join(c.Buckets, " "))
		}
		if err := c.WalkDir(flag.Args()[1]); err != nil {
			color.Warn.Printf("This %s\n", err)
			os.Exit(1)
		}
		if runtime.GOOS == winOS || !c.Quiet {
			fmt.Println(c.Status())
		}
		return
	default:
		color.Warn.Printf("This database command is not valid: '%s'\n", args[0])
	}
}

func taskScan(c *dupers.Config, lookup, quiet bool) {
	l := len(flag.Args())
	const minArgs = 3
	if l < minArgs {
		tsPrintErr(l)
	}
	// directory or a file to match
	c.Source = flag.Args()[1]
	// directories and files to scan, a bucket is the name given to database tables
	c.Buckets = flag.Args()[2:]
	// files or directories to compare (these are not saved to database)
	c.WalkSource()
	// windows notice
	if runtime.GOOS == winOS && !quiet {
		fmt.Printf("To improve performance on Windows use the quiet flag: duper -quiet dupe %s %s\n", c.Source, strings.Join(c.Buckets, " "))
	}
	// walk, scan and save file paths and hashes to the database
	if !lookup {
		if err := database.Clean(true); err != nil {
			log.Println(err)
		}
		c.WalkDirs()
	} else {
		c.Seek()
	}
	// print the found dupes
	c.Print()
	// summaries
	if runtime.GOOS == winOS || !c.Quiet {
		fmt.Println(c.Status())
	}
}

func tsPrintErr(l int) {
	const minArgs = 2
	if l < minArgs {
		color.Warn.Println("the dupe request requires at both a source and target to run a check against")
		fmt.Println("the source can be either a directory or file")
		if runtime.GOOS == winOS {
			fmt.Println("the target can be one or more directories or drive letters")
		} else {
			fmt.Println("the target can be one or more directories")
		}
		fmt.Println("\ndupers dupe <source file or directory> <target one or more directories>")
	}
	if l == minArgs {
		if runtime.GOOS == winOS {
			color.Warn.Println("the dupe request requires at least one target directory or drive letter")
		} else {
			color.Warn.Println("the dupe request requires at least one target directory")
		}
		fmt.Printf("\ndupers dupe %s <target one or more directories>\n", flag.Args()[1])
	}
	fmt.Println("")
	os.Exit(1)
}

func taskSearch(exact, filename, quiet *bool) {
	l := len(flag.Args())
	tscrPrintErr(l)
	term := flag.Args()[1]
	var buckets = []string{}
	const minArgs = 2
	if l > minArgs {
		buckets = flag.Args()[2:]
	}
	var (
		m   *database.Matches
		err error
	)
	if *filename {
		if !*exact {
			if m, err = database.CompareBaseNoCase(term, buckets); err != nil {
				taskSearchErr(term, err)
			}
		}
		if *exact {
			if m, err = database.CompareBase(term, buckets); err != nil {
				taskSearchErr(term, err)
			}
		}
	}
	if !*filename {
		if !*exact {
			if m, err = database.CompareNoCase(term, buckets); err != nil {
				taskSearchErr(term, err)
			}
		}
		if *exact {
			if m, err = database.Compare(term, buckets); err != nil {
				taskSearchErr(term, err)
			}
		}
	}
	dupers.Print(term, *quiet, m)
	if !*quiet {
		fmt.Println(compareResults(len(*m), term, exact, filename))
	}
}

func taskSearchErr(term string, err error) {
	if errors.As(err, &database.ErrNoBucket) {
		color.Warn.Printf("Could not search for '%s'\n", term)
		fmt.Printf("The database %s\n\n", err)
		fmt.Println("To manually add the directory to the database:")
		dir := strings.ReplaceAll(err.Error(), errors.Unwrap(err).Error()+": ", "")
		fmt.Printf("dupers up %s\n", dir)
		os.Exit(1)
	}
	log.Fatalln(err)
}

func tscrPrintErr(l int) {
	if l <= 1 {
		color.Warn.Println("This search request needs an expression")
		fmt.Println("A search expression can be a partial or complete filename,")
		fmt.Println("or a partial or complete directory.")
		fmt.Println("\ndupers search <search expression> [optional, directories to search]")
		fmt.Println("")
		os.Exit(1)
	}
}

func compareResults(total int, term string, exact, filename *bool) string {
	s, r := "", "results"
	if total == 0 {
		return fmt.Sprintf("No results exist for '%s'\n", term)
	}
	if total == 1 {
		r = "result"
	}

	s = fmt.Sprintf("\n%d", total)
	if *exact && *filename {
		s += fmt.Sprintf(" exact filename %s for '%s'", r, term)
		return s
	}
	if *exact {
		s += fmt.Sprintf(" exact %s for '%s'", r, term)
	} else if *filename {
		s += fmt.Sprintf(" filename %s for '%s'", r, term)
	} else {
		s += fmt.Sprintf(" %s for '%s'", r, term)
	}
	return s
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
		fmt.Printf("path: %s\n", err)
		return
	}
	fmt.Printf("path: %s\n", exe)
}

func home() string {
	h, err := os.UserHomeDir()
	if err != nil {
		if h, err = os.Getwd(); err != nil {
			log.Println(err)
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
