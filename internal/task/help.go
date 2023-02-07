// © Ben Garrett https://github.com/bengarrett/dupers
package task

import (
	"flag"
	"fmt"
	"path/filepath"
	"runtime"
	"text/tabwriter"

	"github.com/bengarrett/dupers/internal/cmd"
	"github.com/gookit/color"
)

// DupeHelp creates the dupe command help.
func DupeHelp(f flag.Flag, w *tabwriter.Writer) {
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
	if flag.Lookup("fast") != nil {
		f = *flag.Lookup("fast")
		fmt.Fprintf(w, "    -%v, -%v\t\t%v\n", f.Name[:1], f.Name, f.Usage)
		f = *flag.Lookup("delete")
		fmt.Fprintf(w, "        -%v\t\t%v\n", f.Name, f.Usage)
		f = *flag.Lookup("delete+")
		fmt.Fprintf(w, "        -%v\t\t%v\n", f.Name, f.Usage)
		f = *flag.Lookup("sensen")
		fmt.Fprintf(w, "        -%v\t\t%v\n", f.Name, f.Usage)
	}
	exampleDupe(w)
}

// Search creates the search command help.
func SearchHelp(f flag.Flag, w *tabwriter.Writer) {
	fmt.Fprintf(w, "\n%s\n  Lookup a file or a directory name in the database.\n",
		color.Primary.Sprint("Search:"))
	fmt.Fprintf(w, "  The <search expression> can be a partial or complete, file or directory name.\n")
	fmt.Fprintln(w, "\n  Usage:")
	fmt.Fprintln(w, "    dupers [options] search <search expression> [optional, buckets to search]")
	fmt.Fprintln(w, "\n  Options:")
	if flag.Lookup("exact") != nil {
		f = *flag.Lookup("exact")
		fmt.Fprintf(w, "    -%v, -%v\t\t%v\n", f.Name[:1], f.Name, f.Usage)
		f = *flag.Lookup("name")
		fmt.Fprintf(w, "    -%v, -%v\t\t%v\n", f.Name[:1], f.Name, f.Usage)
	}
	exampleSearch(w)
}

// DBHelp creates the database command help.
func DBHelp(f flag.Flag, w *tabwriter.Writer) {
	fmt.Fprintf(w, "\n%s\n  View information and run optional maintenance on the internal database.\n",
		color.Primary.Sprint("Database:"))
	fmt.Fprintln(w, "\n  Usage:")
	fmt.Fprintf(w, "    dupers %s\tdisplay statistics and bucket information\n", Database_)
	fmt.Fprintf(w, "    dupers %s\t%s\n", Backup_, "make a copy of the database to: "+cmd.Home())
	fmt.Fprintf(w, "    dupers %s\t%s\n", Clean_, "compact and remove all items in the database that point to missing files")
	fmt.Fprintf(w, "    dupers %s  <bucket>\t%s\n", LS_, "list the hashes and files in the bucket")
	fmt.Fprintf(w, "    dupers %s  <bucket>\t%s\n", Up_, "add or update the bucket to the database")
	fmt.Fprintf(w, "    dupers %s <bucket>\t%s\n", UpPlus_, "add or update the bucket using an archive scan "+
		color.Danger.Sprint("(SLOW)")+
		"\n\tthe scan reads every file archived with known package formats")
	fmt.Fprintf(w, "\n    dupers %s  <bucket>\t%s\n", RM_, "remove the bucket from the database")
	fmt.Fprintf(w, "    dupers %s  <bucket> <new directory>\t%s\n", MV_, "move the bucket to a new directory path")
	fmt.Fprintf(w, "    dupers %s <bucket>\t%s\n", Export_, "export the bucket to a text file in: "+cmd.Home())
	fmt.Fprintf(w, "    dupers %s <export file>\t%s\n", Import_, "import a bucket text file into the database")
	fmt.Fprintln(w, "\nOptions:")
	if flag.Lookup("mono") != nil {
		f = *flag.Lookup("mono")
		fmt.Fprintf(w, "    -%v, -%v\t%v\n", f.Name[:1], f.Name, f.Usage)
		f = *flag.Lookup("quiet")
		fmt.Fprintf(w, "    -%v, -%v\t%v\n", f.Name[:1], f.Name, f.Usage)
		f = *flag.Lookup("debug")
		fmt.Fprintf(w, "    -%v, -%v\t%v\n", f.Name[:1], f.Name, f.Usage)
		f = *flag.Lookup("version")
		fmt.Fprintf(w, "    -%v, -%v\t%v\n", f.Name[:1], f.Name, f.Usage)
	}
	fmt.Fprintf(w, "    -h, %s\tshow this list of options\n", "-help")
}

// exampleDupe creates the examples of the dupe command.
func exampleDupe(w *tabwriter.Writer) *tabwriter.Writer {
	fmt.Fprintln(w, "\n  Examples:")
	fmt.Fprint(w, color.Secondary.Sprint("    # find identical copies of file.txt in the Downloads directory\n"))
	if runtime.GOOS == winOS {
		fmt.Fprint(w, color.Info.Sprintf("    dupers dupe \"%s\" \"%s\"",
			filepath.Join(cmd.Home(), "file.txt"), filepath.Join(cmd.Home(), "Downloads")))
		fmt.Fprint(w,
			color.Secondary.Sprint("\n    # search the database for files in Documents that also exist on drives D: and E:\n"))
		fmt.Fprint(w, color.Info.Sprintf("    dupers dupe \"%s\" %s %s",
			filepath.Join(cmd.Home(), "Documents"), "D:", "E:"))
		fmt.Fprintln(w)
		return w
	}
	fmt.Fprint(w, color.Info.Sprintf("    dupers dupe '%s' '%s'",
		filepath.Join(cmd.Home(), "file.txt"), filepath.Join(cmd.Home(), "Downloads")))
	fmt.Fprint(w, color.Secondary.Sprint("\n    # search for files in Documents that also exist in /var/www\n"))
	fmt.Fprint(w, color.Info.Sprintf("    dupers dupe '%s' '%s'",
		filepath.Join(cmd.Home(), "Documents"), "/var/www"))
	fmt.Fprintln(w)
	return w
}

// exampleSearch creates the examples of the search command.
func exampleSearch(w *tabwriter.Writer) *tabwriter.Writer {
	fmt.Fprintln(w, "\n  Examples:")
	fmt.Fprint(w, color.Secondary.Sprint("    # search for the expression foo in your home directory\n"))
	if runtime.GOOS == winOS {
		fmt.Fprint(w, "    "+color.Info.Sprintf("dupers search \"foo\" \"%s\"", cmd.Home()))
		fmt.Fprint(w, color.Secondary.Sprint("\n    # search for filenames containing .zip\n"))
		fmt.Fprint(w, "    "+color.Info.Sprint("dupers -name search \".zip\""))
		fmt.Fprintln(w)
		return w
	}
	fmt.Fprint(w, "    "+color.Info.Sprintf("dupers search 'foo' '%s'", cmd.Home()))
	fmt.Fprint(w, color.Secondary.Sprint("\n    # search for filenames containing .zip\n"))
	fmt.Fprint(w, "    "+color.Info.Sprint("dupers -name search '.zip'"))
	fmt.Fprintln(w)
	return w
}
