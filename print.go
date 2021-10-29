// © Ben Garrett https://github.com/bengarrett/dupers

// Package dupers is the blazing-fast file duplicate checker and filename search.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"

	"github.com/bengarrett/dupers/database"
	"github.com/bengarrett/dupers/dupe"
	"github.com/bengarrett/dupers/out"
	"github.com/gookit/color"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	_ "embed"
)

const (
	tabPadding  = 4
	description = "Dupers is the blazing-fast file duplicate checker and filename search tool."
)

// logo.txt by sensenstahl
//go:embed logo.txt
var brand string // nolint: gochecknoglobals

// Help, usage and examples.
func help() string {
	b, f := bytes.Buffer{}, flag.Flag{}
	w := tabwriter.NewWriter(&b, 0, 0, tabPadding, ' ', 0)

	defer w.Flush()
	fmt.Fprintf(w, "%s\n", description)
	helpDupe(f, w)
	helpSearch(f, w)
	helpDB(f, w)
	fmt.Fprintln(w)
	return b.String()
}

// helpDupe creates the dupe command help.
func helpDupe(f flag.Flag, w *tabwriter.Writer) {
	fmt.Fprintf(w, "\n%s\n  Scan for duplicate files, matching files that share the identical content.\n",
		color.Primary.Sprint("Dupe:"))
	fmt.Fprintln(w, "  The \"directory or file to check\" is never added to the database.")
	if runtime.GOOS == winOS {
		fmt.Fprintln(w,
			"  The \"buckets to lookup\" are directories or drive letters that get added to the database for quicker scans.")
	} else {
		fmt.Fprintln(w, "  The \"buckets to lookup\" are directories that get added to the database for quicker scans.")
	}
	fmt.Fprintln(w, "\n  Usage:")
	fmt.Fprintln(w, "    dupers [options] dupe <directory or file to check> [buckets to lookup]")
	fmt.Fprintln(w, "\n  Options:")
	f = *flag.Lookup("fast")
	fmt.Fprintf(w, "    -%v, -%v\t\t%v\n", f.Name[:1], f.Name, f.Usage)
	f = *flag.Lookup("delete")
	fmt.Fprintf(w, "        -%v\t\t%v\n", f.Name, f.Usage)
	f = *flag.Lookup("delete+")
	fmt.Fprintf(w, "        -%v\t\t%v\n", f.Name, f.Usage)
	f = *flag.Lookup("sensen")
	fmt.Fprintf(w, "        -%v\t\t%v\n", f.Name, f.Usage)
	exampleDupe(w)
}

// helpSearch creates the search command help.
func helpSearch(f flag.Flag, w *tabwriter.Writer) {
	fmt.Fprintf(w, "\n%s\n  Lookup a file or a directory name in the database.\n",
		color.Primary.Sprint("Search:"))
	fmt.Fprintf(w, "  The <search expression> can be a partial or complete, file or directory name.\n")
	fmt.Fprintln(w, "\n  Usage:")
	fmt.Fprintln(w, "    dupers [options] search <search expression> [optional, buckets to search]")
	fmt.Fprintln(w, "\n  Options:")
	f = *flag.Lookup("exact")
	fmt.Fprintf(w, "    -%v, -%v\t\t%v\n", f.Name[:1], f.Name, f.Usage)
	f = *flag.Lookup("name")
	fmt.Fprintf(w, "    -%v, -%v\t\t%v\n", f.Name[:1], f.Name, f.Usage)
	exampleSearch(w)
}

// helpDB creates the database commands help.
func helpDB(f flag.Flag, w *tabwriter.Writer) {
	fmt.Fprintf(w, "\n%s\n  View information and run optional maintenance on the internal database.\n",
		color.Primary.Sprint("Database:"))
	fmt.Fprintln(w, "\n  Usage:")
	fmt.Fprintf(w, "    dupers %s\tdisplay statistics and bucket information\n", dbf)
	fmt.Fprintf(w, "    dupers %s\t%s\n", dbk, "make a copy of the database to: "+home())
	fmt.Fprintf(w, "    dupers %s\t%s\n", dcn, "compact and remove all items in the database that point to missing files")
	fmt.Fprintf(w, "    dupers %s  <bucket>\t%s\n", dls, "list the hashes and files in the bucket")
	fmt.Fprintf(w, "    dupers %s  <bucket>\t%s\n", dup, "add or update the bucket to the database")
	fmt.Fprintf(w, "    dupers %s <bucket>\t%s\n", dupp, "add or update the bucket using an archive scan "+
		color.Danger.Sprint("(SLOW)")+
		"\n\tthe scan reads all the files stored within file archives")
	fmt.Fprintf(w, "\n    dupers %s  <bucket>\t%s\n", drm, "remove the bucket from the database")
	fmt.Fprintf(w, "    dupers %s  <bucket> <new directory>\t%s\n", dmv, "move the bucket to a new directory path")
	fmt.Fprintf(w, "    dupers %s <bucket>\t%s\n", dex, "export the bucket to a text file in: "+home())
	fmt.Fprintf(w, "    dupers %s <export file>\t%s\n", dim, "import a bucket text file into the database")
	fmt.Fprintln(w, "\nOptions:")
	f = *flag.Lookup("mono")
	fmt.Fprintf(w, "    -%v, -%v\t%v\n", f.Name[:1], f.Name, f.Usage)
	f = *flag.Lookup("quiet")
	fmt.Fprintf(w, "    -%v, -%v\t%v\n", f.Name[:1], f.Name, f.Usage)
	f = *flag.Lookup("version")
	fmt.Fprintf(w, "    -%v, -%v\t%v\n", f.Name[:1], f.Name, f.Usage)
	fmt.Fprintf(w, "    -h, %s\tshow this list of options\n", fhlp)
}

// exampleDupe creates the dupe command examples.
func exampleDupe(w *tabwriter.Writer) *tabwriter.Writer {
	fmt.Fprintln(w, "\n  Examples:")
	fmt.Fprint(w, color.Secondary.Sprint("    # find identical copies of file.txt in the Downloads directory\n"))
	if runtime.GOOS == winOS {
		fmt.Fprint(w, color.Info.Sprintf("    dupers dupe \"%s\" \"%s\"",
			filepath.Join(home(), "file.txt"), filepath.Join(home(), "Downloads")))
		fmt.Fprint(w,
			color.Secondary.Sprint("\n    # search the database for files in Documents that also exist on drives D: and E:\n"))
		fmt.Fprint(w, color.Info.Sprintf("    dupers dupe \"%s\" %s %s",
			filepath.Join(home(), "Documents"), "D:", "E:"))
		fmt.Fprintln(w)
		return w
	}
	fmt.Fprint(w, color.Info.Sprintf("    dupers dupe '%s' '%s'",
		filepath.Join(home(), "file.txt"), filepath.Join(home(), "Downloads")))
	fmt.Fprint(w, color.Secondary.Sprint("\n    # search for files in Documents that also exist in /var/www\n"))
	fmt.Fprint(w, color.Info.Sprintf("    dupers dupe '%s' '%s'",
		filepath.Join(home(), "Documents"), "/var/www"))
	fmt.Fprintln(w)
	return w
}

// exampleSearch creates the example command examples.
func exampleSearch(w *tabwriter.Writer) *tabwriter.Writer {
	fmt.Fprintln(w, "\n  Examples:")
	fmt.Fprint(w, color.Secondary.Sprint("    # search for the expression foo in your home directory\n"))
	if runtime.GOOS == winOS {
		fmt.Fprint(w, "    "+color.Info.Sprintf("dupers search \"foo\" \"%s\"", home()))
		fmt.Fprint(w, color.Secondary.Sprint("\n    # search for filenames containing .zip\n"))
		fmt.Fprint(w, "    "+color.Info.Sprint("dupers -name search \".zip\""))
		fmt.Fprintln(w)
		return w
	}
	fmt.Fprint(w, "    "+color.Info.Sprintf("dupers search 'foo' '%s'", home()))
	fmt.Fprint(w, color.Secondary.Sprint("\n    # search for filenames containing .zip\n"))
	fmt.Fprint(w, "    "+color.Info.Sprint("dupers -name search '.zip'"))
	fmt.Fprintln(w)
	return w
}

// checkDupePaths checks the path arguments supplied to the dupe command.
func checkDupePaths(c *dupe.Config) {
	if ok, cc, bc := c.CheckPaths(); !ok {
		p := message.NewPrinter(language.English)
		verb := "Buckets"
		if len(c.Buckets()) == 1 {
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
}

// vers prints out the program information and version.
func vers() string {
	const copyright, year = "\u00A9", 2021
	exe, err := self()
	if err != nil {
		out.ErrCont(err)
	}
	w := new(bytes.Buffer)
	fmt.Fprintln(w, brand+"\n")
	fmt.Fprintf(w, "                                dupers v%s\n", version)
	fmt.Fprintf(w, "                           %s %d Ben Garrett\n", copyright, year)
	fmt.Fprintf(w, "         %s\n\n", color.Primary.Sprint("https://github.com/bengarrett/dupers"))
	fmt.Fprintf(w, "  %s    %s (%s)\n", color.Secondary.Sprint("build:"), commit, date)
	fmt.Fprintf(w, "  %s %s/%s\n", color.Secondary.Sprint("platform:"), runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(w, "  %s       %s\n", color.Secondary.Sprint("go:"), strings.Replace(runtime.Version(), "go", "v", 1))
	fmt.Fprintf(w, "  %s     %s\n", color.Secondary.Sprint("path:"), exe)
	return w.String()
}

// dupeCmdErr parses the arguments of the dupe command.
func dupeCmdErr(args, buckets, minArgs int) {
	if args < minArgs {
		out.ErrCont(ErrNoArgs)
		fmt.Println("\nThe dupe command requires a directory or file to check.")
		if runtime.GOOS == winOS {
			fmt.Println("The optional bucket can be one or more directories or drive letters.")
		} else {
			fmt.Println("The optional bucket can be one or more directory paths.")
		}
		out.Example("\ndupers dupe <directory or file to check> [buckets to lookup]")
	}
	if buckets == 0 && args == minArgs {
		color.Warn.Println("The database is empty.\n")
		if runtime.GOOS == winOS {
			fmt.Println("This dupe request requires at least one directory or drive letter to lookup.")
		} else {
			fmt.Println("This dupe request requires at least one directory to lookup.")
		}
		fmt.Println("These lookup directories will be stored to the database as buckets.")
		s := fmt.Sprintf("\ndupers dupe %s <one or more directories>\n", flag.Args()[1])
		out.Example(s)
	}
	out.ErrFatal(nil)
}

// searchCmdErr parses the arguments of the search command.
func searchCmdErr(l int) {
	if l <= 1 {
		out.ErrCont(ErrSearch)
		fmt.Println("A search expression can be a partial or complete filename,")
		fmt.Println("or a partial or complete directory.")
		out.Example("\ndupers search <search expression> [optional, directories to search]")
		out.ErrFatal(nil)
	}
}

// searchErr parses the errors from search compares.
func searchErr(err error) {
	if errors.Is(err, database.ErrDBEmpty) {
		out.ErrCont(err)
		return
	}
	if errors.As(err, &database.ErrBucketNotFound) {
		out.ErrCont(err)
		fmt.Println("\nTo add this directory to the database, run:")
		dir := err.Error()
		if errors.Unwrap(err) == nil {
			s := fmt.Sprintf("%s: ", errors.Unwrap(err))
			dir = strings.ReplaceAll(err.Error(), s, "")
		}
		s := fmt.Sprintf("dupers up %s\n", dir)
		out.Example(s)
		out.ErrFatal(nil)
	}
	out.ErrFatal(err)
}

// Home returns the user's home directory.
// Or if that fails, returns the current working directory.
func home() string {
	h, err := os.UserHomeDir()
	if err != nil {
		if h, err = os.Getwd(); err != nil {
			out.ErrCont(err)
		}
	}
	return h
}

// Self returns the path to this dupers executable file.
func self() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("self error: %w", err)
	}
	return exe, nil
}
