// © Ben Garrett https://github.com/bengarrett/dupers

// BBStruction todo
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"text/tabwriter"

	"github.com/gookit/color"
)

var (
	brand string // nolint: gochecknoglobals

	version = "0.0.0"
	commit  = "unset" // nolint: gochecknoglobals
	date    = "unset" // nolint: gochecknoglobals
)

const winOS = "windows"

func main() {

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
			color.Warn.Println("zipcmt requires at least one directory or drive letter to scan")
		} else {
			color.Warn.Println("zipcmt requires at least one directory to scan")
		}
		fmt.Println()
		help(false)
		os.Exit(0)
	}
}

// Help, usage and examples.
func help(logo bool) {
	var f *flag.Flag
	const ps = string(os.PathSeparator)
	if logo {
		fmt.Fprintln(os.Stderr, brand)
	}
	fmt.Fprintln(os.Stderr, "Usage:")
	if runtime.GOOS == winOS {
		fmt.Fprintln(os.Stderr, "    zipcmt [options] <directories or drive letters>")
	} else {
		fmt.Fprintln(os.Stderr, "    zipcmt [options] <directories>")
	}
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Examples:")
}

// Info prints out the program information and version.
func info() {
	const copyright = "\u00A9"
	fmt.Println(brand)
	fmt.Printf("zipcmt v%s\n%s 2021 Ben Garrett, logo by sensenstahl\n", version, copyright)
	fmt.Printf("https://github.com/bengarrett/zipcmt\n\n")
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
