// © Ben Garrett https://github.com/bengarrett/dupers

// dupers todo
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"text/tabwriter"
	"time"

	dupers "github.com/bengarrett/dupers/lib"
	"github.com/bengarrett/dupers/lib/database"
	"github.com/dustin/go-humanize"
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

	flag.BoolVar(&c.Lookup, "fast", false, "query the database for a much faster match,\n\t\tthe results maybe stale as it does not look for any file or directory changes")
	f := flag.Bool("f", false, "alias for fast")
	exact := flag.Bool("exact", false, "match case")
	ex := flag.Bool("e", false, "alias for exact")
	filename := flag.Bool("name", false, "search for filenames and ignore directories")
	fn := flag.Bool("n", false, "alias for name")
	ver := flag.Bool("version", false, "version and information for this program")
	v := flag.Bool("v", false, "alias for version")

	flag.Usage = func() {
		help(true)
	}
	flag.Parse()
	options(ver, v)

	selection := strings.ToLower(flag.Args()[0])
	switch selection {
	case "database", "db":
		taskDatabase(flag.Args()...)
	case "dupe":
		if *f {
			c.Lookup = true
		}
		taskScan(c)
	case "search":
		if *ex {
			exact = ex
		}
		if *fn {
			filename = fn
		}
		taskSearch(exact, filename)
	}
}

// Help, usage and examples.
func help(logo bool) {
	var f *flag.Flag
	w := tabwriter.NewWriter(os.Stderr, 0, 0, 4, ' ', 0)
	//const ps = string(os.PathSeparator)
	if logo {
		fmt.Fprintln(os.Stderr, brand)
	}
	fmt.Fprintln(w, "Dupe:\n  Scan for duplicate files.")
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

	fmt.Fprintln(w, "\nSearch:\n  Lookup file and directory names in the database.")
	fmt.Fprintln(w, "\n  Usage:")
	fmt.Fprintln(w, "    dupers [options] search <search name> [directories to search]")
	fmt.Fprintln(w, "\n  Options:")
	f = flag.Lookup("exact")
	fmt.Fprintf(w, "    -%v, -%v\t\t%v\n", f.Name[:1], f.Name, f.Usage)
	f = flag.Lookup("name")
	fmt.Fprintf(w, "    -%v, -%v\t\t%v\n", f.Name[:1], f.Name, f.Usage)
	fmt.Fprintln(w, "\n  Examples:")

	fmt.Fprintln(w, "\nDatabase:\n  View information and run maintenance on the internal database.\n  All of these commands are optional and are not required for the normal usage of dupers.")
	fmt.Fprintln(w, "\n  Usage:")
	fmt.Fprintln(w, "    dupers database\tdisplay statistics and bucket information")
	fmt.Fprintf(w, "    dupers database %s\t%s\n", "backup", "make a copy of the database to: "+home())
	fmt.Fprintf(w, "    dupers database %s\t%s\n", "clean", "remove all items in the database that point to missing files")
	fmt.Fprintf(w, "    dupers database %s <bucket>\t%s\n", "rm", "remove the bucket (a scanned directory path) from the database")

	fmt.Fprintln(w, "\nOptions:")
	f = flag.Lookup("version")
	fmt.Fprintf(w, "    -%v, -%v\t%v\n", f.Name[:1], f.Name, f.Usage)
	fmt.Fprintln(w, "    -h, -help\tshow this list of options")
	fmt.Fprintln(w)
	optimial(w)
	w.Flush()
}

func taskDatabase(args ...string) {
	// handle any database tasks and exit
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
		fmt.Println("saved a copy of the database", humanize.Bytes(uint64(w)), ":", n)
		return
	case "clean":
		if err := database.Clean(); err != nil {
			log.Fatalln(err)
		}
		return
	case "rm":
		if l == 2 {
			fmt.Println("no bucket was provided")
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

func taskScan(c dupers.Config) {
	// if runtime.GOOS == winOS {
	// 	color.Warn.Println("dupers requires at least one directory or drive letter to scan")
	// } else {
	// 	color.Warn.Println("dupers requires at least one directory to dupe")
	// }
	fmt.Println()
	l := len(flag.Args())
	if l <= 1 {
		fmt.Println("dupe requires both a directory/file to match and one or more directories to lookup")
		os.Exit(1)
	}
	if l == 2 {
		fmt.Println("dupe requires at least one directory to lookup")
		// todo show example with user params
		os.Exit(1)
	}
	// directory or a file to match
	c.Bucket = flag.Args()[1]
	// directories and files to scan, a bucket is the name given to database tables
	c.Buckets = flag.Args()[2:]
	// files or directories to compare (these are not saved to database)
	c.Queries()
	// walk, scan and save file paths and hashes to the database
	if c.Lookup {
		c.Query()
	} else {
		if err := database.Clean(); err != nil {
			log.Println(err)
		}
		c.WalkDirs()
	}
	fmt.Println()
	// find dupes
	c.Matches()
	// summaries
	fmt.Println(c.Status())
}

func taskSearch(exact, filename *bool) {
	fmt.Println()
	l := len(flag.Args())
	if l <= 1 {
		fmt.Println("search requires a <search name> which can be a partial or complete filename or directory")
		os.Exit(1)
	}
	term := flag.Args()[1]
	var buckets = []string{}
	if l > 2 {
		buckets = flag.Args()[2:]
	}
	if *filename {
		if !*exact {
			database.CompareBaseNoCase(term, buckets)
			return
		}
		database.CompareBase(term, buckets)
		return
	}
	if !*exact {
		database.CompareNoCase(term, buckets)
		return
	}
	database.Compare(term, buckets)
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
