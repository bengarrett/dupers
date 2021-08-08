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
	"github.com/bengarrett/dupers/dupers"
	"github.com/bengarrett/dupers/out"
	"github.com/gookit/color"
)

const tabPadding = 4

// Help, usage and examples.
func help() string {
	b, f := bytes.Buffer{}, flag.Flag{}
	w := tabwriter.NewWriter(&b, 0, 0, tabPadding, ' ', 0)
	defer w.Flush()
	fmt.Fprintf(w, "Dupers is the blazing-fast file duplicate checker and filename search.\n")
	helpDupe(f, w)
	helpSearch(f, w)
	helpDB(f, w)
	fmt.Fprintln(w)
	return b.String()
}

func helpDupe(f flag.Flag, w *tabwriter.Writer) {
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
	fmt.Fprintf(w, "    dupers %s <export file>\t%s\n", dim, "import the bucket text file into the database")
	fmt.Fprintln(w, "\nOptions:")
	f = *flag.Lookup("quiet")
	fmt.Fprintf(w, "    -%v, -%v\t%v\n", f.Name[:1], f.Name, f.Usage)
	f = *flag.Lookup("version")
	fmt.Fprintf(w, "    -%v, -%v\t%v\n", f.Name[:1], f.Name, f.Usage)
	fmt.Fprintln(w, "    -h, -help\tshow this list of options")
}

func exampleDupe(w *tabwriter.Writer) *tabwriter.Writer {
	fmt.Fprintln(w, "\n  Examples:")
	fmt.Fprint(w, color.Secondary.Sprint("    # find identical copies of file.txt in the Downloads directory\n"))
	if runtime.GOOS == winOS {
		fmt.Fprint(w, color.Info.Sprintf("    dupers dupe \"%s\" \"%s\"",
			filepath.Join(home(), "file.txt"), filepath.Join(home(), "Downloads")))
		fmt.Fprint(w, color.Secondary.Sprint("\n    # search for files in Documents that also exist on drives D: and E:\n"))
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

// Info prints out the program information and version.
func info() string {
	const copyright = "\u00A9"
	exe, err := self()
	if err != nil {
		out.ErrCont(err)
	}
	var w = new(bytes.Buffer)
	fmt.Fprintf(w, "dupers v%s\n%s 2021 Ben Garrett\n", version, copyright)
	fmt.Fprintf(w, "https://github.com/bengarrett/dupers\n\n")
	fmt.Fprintf(w, "build: %s (%s)\n", commit, date)
	fmt.Fprintf(w, "go:    %s\n", strings.Replace(runtime.Version(), "go", "v", 1))
	fmt.Fprintf(w, "path:  %s\n", exe)
	return w.String()
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

func taskCheckPaths(c *dupers.Config) {
	if ok, cc, bc := c.CheckPaths(); !ok {
		fmt.Printf("Directory to check:                  %s (%s)\n", c.ToCheck(), color.Red.Sprintf("%d files", cc))
		fmt.Printf("Buckets to lookup in for duplicates: %s (%d files)\n\n", c.PrintBuckets(), bc)
		color.Notice.Println("Please confirm the directories are correct.")
		color.Info.Println("The dictory to check is not stored to the database.")
		if !out.YN("Is this what you want") {
			os.Exit(0)
		}
	}
}

func searchCmdErr(l int) {
	if l <= 1 {
		out.ErrCont(ErrSearch)
		fmt.Println("A search expression can be a partial or complete filename,")
		fmt.Println("or a partial or complete directory.")
		out.Example("\ndupers search <search expression> [optional, directories to search]")
		out.ErrFatal(nil)
	}
}

func dupeCmdErr(args, buckets int) {
	const minArgs = 2
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

func taskSearchErr(err error) {
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
