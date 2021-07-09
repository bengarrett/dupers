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
	brand string // nolint: gochecknoglobals

	version = "0.0.0"
	commit  = "unset" // nolint: gochecknoglobals
	date    = "unset" // nolint: gochecknoglobals
)

const winOS = "windows"

// --delete-dupes
// --move-dupes
// --copy-dupes
//
// --deep-scan (open archives and hash binary files)

func main() {
	var c dupers.Config
	c.Timer = time.Now()

	look := flag.Bool("fast", false, "query the database for a much faster match,\n\t\tthe results maybe stale as it does not look for any file or directory changes")
	f := flag.Bool("f", false, "alias for fast")
	exact := flag.Bool("exact", false, "match case")
	ex := flag.Bool("e", false, "alias for exact")
	filename := flag.Bool("name", false, "search for filenames and ignore directories")
	fn := flag.Bool("n", false, "alias for name")

	quiet := flag.Bool("quiet", false, "quiet mode hides all but essential feedback")
	q := flag.Bool("q", false, "alias for quiet")
	ver := flag.Bool("version", false, "version and information for this program")
	v := flag.Bool("v", false, "alias for version")

	flag.Usage = func() {
		help(true)
	}
	flag.Parse()
	if *q {
		*quiet = true
	}
	options(ver, v)

	selection := strings.ToLower(flag.Args()[0])
	switch selection {
	case "database", "db":
		taskDatabase(*quiet, flag.Args()...)
	case "dupe":
		if *f {
			*look = true
		}
		if *q || *quiet {
			c.Quiet = true
		}
		taskScan(c, *look)
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
func help(logo bool) {
	var f *flag.Flag
	w := tabwriter.NewWriter(os.Stderr, 0, 0, 4, ' ', 0)
	if logo {
		fmt.Fprintln(os.Stderr, brand)
	}
	fmt.Fprintln(w, "Dupe:\n  Scan for duplicate files, looking for files that share identical content.")
	fmt.Fprintln(w, "  The \"directory or file to match\" is never added to the database.")
	fmt.Fprintln(w, "  The contents of the \"directories to look up\" will always be added to the database for quicker, future scans.")
	fmt.Fprintln(w, "\n  Usage:")
	if runtime.GOOS == winOS {
		fmt.Fprintln(w, "    dupers [options] dupe <directory or file to match> <directories or drive letters to lookup>")
	} else {
		fmt.Fprintln(w, "    dupers [options] dupe <directory or file to match> <directories to lookup>")
	}
	fmt.Fprintln(w, "\n  Options:")
	f = flag.Lookup("fast")
	fmt.Fprintf(w, "    -%v, -%v\t\t%v\n", f.Name[:1], f.Name, f.Usage)
	w.Flush()
	fmt.Fprintln(w, "\n  Examples:")
	fmt.Fprint(w, color.Info.Sprintf("    dupers dupe %q %q",
		filepath.Join(home(), "file.zip"), filepath.Join(home(), "Downloads")))
	fmt.Fprint(w, color.Note.Sprint("\t# search for identical copies of file.zip in the Downloads directory\n"))
	fmt.Fprint(w, color.Info.Sprintf("    dupers -fast dupe %q %q",
		"doc.txt", filepath.Join(home(), "Documents")))
	fmt.Fprint(w, color.Note.Sprint("\t\t# search the database for identical copies of doc.txt in the Documents directory\n"))

	fmt.Fprintln(w, "\nSearch:\n  Lookup a file or a directory name in the database.")
	fmt.Fprintln(w, "\n  Usage:")
	fmt.Fprintln(w, "    dupers [options] search <search expression> [directories to search]")
	fmt.Fprintln(w, "\n  Options:")
	f = flag.Lookup("exact")
	fmt.Fprintf(w, "    -%v, -%v\t\t%v\n", f.Name[:1], f.Name, f.Usage)
	f = flag.Lookup("name")
	fmt.Fprintf(w, "    -%v, -%v\t\t%v\n", f.Name[:1], f.Name, f.Usage)
	fmt.Fprintln(w, "\n  Examples:")
	fmt.Fprint(w, "    "+color.Info.Sprintf("dupers search \"foo\" %q", home()))
	fmt.Fprint(w, color.Note.Sprint("\t# search for the expression foo in your home directory\n"))
	fmt.Fprint(w, "    "+color.Info.Sprint("dupers search \"bar\""))
	fmt.Fprint(w, color.Note.Sprint("\t\t# search for the expression bar in the database\n"))
	fmt.Fprint(w, "    "+color.Info.Sprint("dupers -name search \".zip\""))
	fmt.Fprint(w, color.Note.Sprint("\t\t# search for filenames containing .zip\n"))

	fmt.Fprintln(w, "\nDatabase:\n  View information and run maintenance on the internal database.\n  All of these commands are optional and are not required for the normal usage of dupers.")
	fmt.Fprintln(w, "\n  Usage:")
	fmt.Fprintln(w, "    dupers database\tdisplay statistics and bucket information")
	fmt.Fprintf(w, "    dupers database %s\t%s\n", "backup", "make a copy of the database to: "+home())
	fmt.Fprintf(w, "    dupers database %s\t%s\n", "clean", "remove all items in the database that point to missing files")
	fmt.Fprintf(w, "    dupers database %s <bucket>\t%s\n", "rm", "remove the bucket (a scanned directory path) from the database")

	fmt.Fprintln(w, "\nOptions:")
	f = flag.Lookup("quiet")
	fmt.Fprintf(w, "    -%v, -%v\t%v\n", f.Name[:1], f.Name, f.Usage)
	f = flag.Lookup("version")
	fmt.Fprintf(w, "    -%v, -%v\t%v\n", f.Name[:1], f.Name, f.Usage)
	fmt.Fprintln(w, "    -h, -help\tshow this list of options")
	fmt.Fprintln(w)
	optimial(w)
	w.Flush()
}

func taskDatabase(quiet bool, args ...string) {
	l := len(args)
	if l < 2 {
		fmt.Println(database.Info())
		return
	}
	switch args[1] {
	case "backup":
		n, w, err := database.Backup()
		if err != nil {
			log.Fatalln(err)
		}
		if !quiet {
			fmt.Println("saved a copy of the database", humanize.Bytes(uint64(w)), ":", n)
		}
		return
	case "clean":
		if err := database.Clean(quiet); err != nil {
			log.Fatalln(err)
		}
		return
	case "rm":
		if l == 2 {
			color.Warn.Println("Cannot remove a bucket from the database as no bucket name was provided")
			fmt.Println("\ndupers database rm <bucket name>")
			fmt.Println()
			os.Exit(1)
		}
		name := args[2]
		if err := database.RM(name); err != nil {
			if errors.Is(err, database.ErrNoBucket) {
				fmt.Printf("The bucket does not exist in the database: %q\n", name)
				buckets, err := database.Buckets()
				if err != nil {
					log.Fatalln(err)
				}
				fmt.Printf("Buckets in use: %s\n", strings.Join(buckets, "\n\t\t"))
				os.Exit(1)
			}
			log.Fatalln(err)
		}
		fmt.Printf("Removed bucket from the database: %q\n", name)
		return
	}
}

func taskScan(c dupers.Config, lookup bool) {
	l := len(flag.Args())
	if l < 3 {
		tsPrintErr(l)
	}
	// directory or a file to match
	c.Source = flag.Args()[1]
	// directories and files to scan, a bucket is the name given to database tables
	c.Buckets = flag.Args()[2:]
	// files or directories to compare (these are not saved to database)
	c.WalkSource()
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
	if !c.Quiet {
		fmt.Println(c.Status())
	}
}

func tsPrintErr(l int) {
	if l < 2 {
		color.Warn.Println("the dupe request requires at both a source and target to run a check against")
		fmt.Println("the source can be either a directory or file")
		if runtime.GOOS == winOS {
			fmt.Println("the target can be one or more directories or drive letters")
		} else {
			fmt.Println("the target can be one or more directories")
		}
		fmt.Println("\ndupers dupe <source file or directory> <target one or more directories>")
	}
	if l == 2 {
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
	if l > 2 {
		buckets = flag.Args()[2:]
	}
	var (
		m   *database.Matches
		err error
	)
	if *filename {
		if !*exact {
			if m, err = database.CompareBaseNoCase(term, buckets); err != nil {
				log.Fatalln(err)
			}
		}
		if *exact {
			if m, err = database.CompareBase(term, buckets); err != nil {
				log.Fatalln(err)
			}
		}
	}
	if !*filename {
		if !*exact {
			if m, err = database.CompareNoCase(term, buckets); err != nil {
				log.Fatalln(err)
			}
		}
		if *exact {
			if m, err = database.Compare(term, buckets); err != nil {
				log.Fatalln(err)
			}
		}
	}
	dupers.Print(term, *quiet, m)
	if !*quiet {
		fmt.Println(compareResults(len(*m), term, exact, filename))
	}
}

func tscrPrintErr(l int) {
	if l <= 1 {
		color.Warn.Println("the search request needs an expression")
		fmt.Println("a search expression can be a partial or complete filename,")
		fmt.Println("or a partial or complete directory")
		fmt.Println("\ndupers search <search expression> [optional, directories to search]")
		fmt.Println("")
		os.Exit(1)
	}
}

func compareResults(total int, term string, exact, filename *bool) string {
	s, r := "", "results"
	if total == 0 {
		return fmt.Sprintf("No results exist for %q\n", term)
	}
	if total == 1 {
		r = "result"
	}

	s = fmt.Sprintf("\n%d", total)
	if *exact && *filename {
		s += fmt.Sprintf(" exact filename %s for %q", r, term)
		return s
	}
	if *exact {
		s += fmt.Sprintf(" exact %s for %q", r, term)
	} else if *filename {
		s += fmt.Sprintf(" filename %s for %q", r, term)
	} else {
		s += fmt.Sprintf(" %s for %q", r, term)
	}
	return s
}

func options(ver, v *bool) {
	// convenience for when a help or version flag is passed as an argument
	for _, arg := range flag.Args() {
		switch strings.ToLower(arg) {
		case "-h", "-help", "--help":
			help(true)
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
		help(false)
		os.Exit(0)
	}
}

// Info prints out the program information and version.
func info() {
	const copyright = "\u00A9"
	fmt.Println(brand)
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

func optimial(w *tabwriter.Writer) {
	if runtime.GOOS == winOS {
		fmt.Fprintln(w, "For optimal performance Windows users may wish to temporarily disable"+
			" the Virus & threat 'Real-time protection' under Windows Security.")
		fmt.Fprintln(w, "Or create Windows Security Exclusions for the directories to be scanned.")
		fmt.Fprintln(w, "https://support.microsoft.com/en-us/windows/add-an-exclusion-to-windows-security-811816c0-4dfd-af4a-47e4-c301afe13b26")
	}
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
