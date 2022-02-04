package help

import (
	"flag"
	"fmt"
	"path/filepath"
	"runtime"
	"text/tabwriter"

	"github.com/bengarrett/dupers/internal/cmd"
	"github.com/gookit/color"
)

const (
	dbf   = "database"
	dbk   = "backup"
	dcn   = "clean"
	dex   = "export"
	dim   = "import"
	dls   = "ls"
	dmv   = "mv"
	drm   = "rm"
	dup   = "up"
	dupp  = "up+"
	fhlp  = "-help"
	winOS = "windows"
)

// Dupe creates the dupe command help.
func Dupe(f flag.Flag, w *tabwriter.Writer) {
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

// helpSearch creates the search command help.
func Search(f flag.Flag, w *tabwriter.Writer) {
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

// DB creates the database commands help.
func DB(f flag.Flag, w *tabwriter.Writer) {
	fmt.Fprintf(w, "\n%s\n  View information and run optional maintenance on the internal database.\n",
		color.Primary.Sprint("Database:"))
	fmt.Fprintln(w, "\n  Usage:")
	fmt.Fprintf(w, "    dupers %s\tdisplay statistics and bucket information\n", dbf)
	fmt.Fprintf(w, "    dupers %s\t%s\n", dbk, "make a copy of the database to: "+cmd.Home())
	fmt.Fprintf(w, "    dupers %s\t%s\n", dcn, "compact and remove all items in the database that point to missing files")
	fmt.Fprintf(w, "    dupers %s  <bucket>\t%s\n", dls, "list the hashes and files in the bucket")
	fmt.Fprintf(w, "    dupers %s  <bucket>\t%s\n", dup, "add or update the bucket to the database")
	fmt.Fprintf(w, "    dupers %s <bucket>\t%s\n", dupp, "add or update the bucket using an archive scan "+
		color.Danger.Sprint("(SLOW)")+
		"\n\tthe scan reads every file archived with known package formats")
	fmt.Fprintf(w, "\n    dupers %s  <bucket>\t%s\n", drm, "remove the bucket from the database")
	fmt.Fprintf(w, "    dupers %s  <bucket> <new directory>\t%s\n", dmv, "move the bucket to a new directory path")
	fmt.Fprintf(w, "    dupers %s <bucket>\t%s\n", dex, "export the bucket to a text file in: "+cmd.Home())
	fmt.Fprintf(w, "    dupers %s <export file>\t%s\n", dim, "import a bucket text file into the database")
	fmt.Fprintln(w, "\nOptions:")
	if flag.Lookup("mono") != nil {
		f = *flag.Lookup("mono")
		fmt.Fprintf(w, "    -%v, -%v\t%v\n", f.Name[:1], f.Name, f.Usage)
		f = *flag.Lookup("quiet")
		fmt.Fprintf(w, "    -%v, -%v\t%v\n", f.Name[:1], f.Name, f.Usage)
		f = *flag.Lookup("version")
		fmt.Fprintf(w, "    -%v, -%v\t%v\n", f.Name[:1], f.Name, f.Usage)
	}
	fmt.Fprintf(w, "    -h, %s\tshow this list of options\n", fhlp)
}

// exampleDupe creates the dupe command examples.
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

// exampleSearch creates the example command examples.
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
