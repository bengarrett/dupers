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

	dupers "github.com/bengarrett/dupers/lib"
	"github.com/bengarrett/dupers/lib/database"
	"github.com/bengarrett/dupers/lib/out"
	"github.com/gookit/color"
)

// Help, usage and examples.
func help() string {
	var f *flag.Flag
	var b bytes.Buffer
	w := tabwriter.NewWriter(&b, 0, 0, 4, ' ', 0)
	defer w.Flush()
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
	return b.String()
}

func exampleDupe(w *tabwriter.Writer) *tabwriter.Writer {
	fmt.Fprintln(w, "\n  Examples:")
	fmt.Fprint(w, color.Secondary.Sprint("    # find identical copies of file.txt in the Downloads directory\n"))
	fmt.Fprint(w, color.Info.Sprintf("    dupers dupe '%s' '%s'",
		filepath.Join(home(), "file.txt"), filepath.Join(home(), "Downloads")))

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

func taskExpErr(l int) {
	if l <= 1 {
		out.ErrCont(ErrSearch)
		fmt.Println("A search expression can be a partial or complete filename,")
		fmt.Println("or a partial or complete directory.")
		out.Example("\ndupers search <search expression> [optional, directories to search]")
		out.ErrFatal(nil)
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