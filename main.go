// © Ben Garrett https://github.com/bengarrett/dupers

// dupers todo
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"text/tabwriter"
	"time"

	dupers "github.com/bengarrett/dupers/lib"
	"github.com/gookit/color"
)

var (
	brand string // nolint: gochecknoglobals

	version = "0.0.0"
	commit  = "unset" // nolint: gochecknoglobals
	date    = "unset" // nolint: gochecknoglobals
)

const winOS = "windows"

// scan folders & files
// search location
//
// archives/collection location
// scan location (where to look)
//
// --scan [file/directory] // ~/zipcmt.deb
//
// --delete-dupes
// --move-dupes
// --copy-dupes
//
// --deep-scan (open archives and hash binary files)
//
//
// dupers --scan=~/zipcmt.deb ~
// dupers --scan=C:\Users\ben\Downloads\zipcmt.deb C:\Users\ben

func main() {
	var c dupers.Config
	c.Timer = time.Now()
	flag.StringVar(&c.Scan, "scan", "", "scan this file or directory")

	ver := flag.Bool("version", false, "version and information for this program")
	v := flag.Bool("v", false, "alias for version")

	flag.Usage = func() {
		help(true)
	}
	flag.Parse()
	flags(ver, v)
	// directories to scan
	c.Dirs = flag.Args()
	// file and directory scan
	c.WalkDirs()
	// find dupes
	c.ScanDirs()
	// summaries
	fmt.Println(c.Status())
}

func flags(ver, v *bool) {
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
		if runtime.GOOS == winOS {
			color.Warn.Println("dupers requires at least one directory or drive letter to scan")
		} else {
			color.Warn.Println("dupers requires at least one directory to scan")
		}
		fmt.Println()
		help(false)
		os.Exit(0)
	}
}

// Help, usage and examples.
func help(logo bool) {
	var f *flag.Flag
	//const ps = string(os.PathSeparator)
	if logo {
		fmt.Fprintln(os.Stderr, brand)
	}
	fmt.Fprintln(os.Stderr, "Usage:")
	if runtime.GOOS == winOS {
		fmt.Fprintln(os.Stderr, "    dupers [options] <directories or drive letters>")
	} else {
		fmt.Fprintln(os.Stderr, "    dupers [options] <directories>")
	}
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Examples:")
	// todo
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Options:")
	w := tabwriter.NewWriter(os.Stderr, 0, 0, 4, ' ', 0)
	f = flag.Lookup("scan")
	fmt.Fprintf(w, "    -%v, -%v=<DIRECTORY|FILE>\t%v\n", "s", f.Name, f.Usage)
	f = flag.Lookup("version")
	fmt.Fprintf(w, "    -%v, -%v\t%v\n", f.Name[:1], f.Name, f.Usage)
	fmt.Fprintln(w, "    -h, -help\tshow this list of options")
	fmt.Fprintln(w)
	optimial(w)
	w.Flush()
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

func self() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("self error: %w", err)
	}
	return exe, nil
}
