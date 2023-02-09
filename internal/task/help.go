// © Ben Garrett https://github.com/bengarrett/dupers
package task

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"text/tabwriter"

	"github.com/bengarrett/dupers/internal/cmd"
	"github.com/gookit/color"
)

func Debug(a *cmd.Aliases, f *cmd.Flags) string {
	const na = "n/a"
	buf := new(bytes.Buffer)
	w := tabwriter.NewWriter(buf, 0, 0, 1, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, "Dupers arguments debug:")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "\t\tToggle\t\tAlias")
	fmt.Fprintf(w, "-mono:\t\t%v\t\t%v\n", *f.Mono, *a.Mono)
	fmt.Fprintf(w, "-quiet:\t\t%v\t\t%v\n", *f.Quiet, *a.Quiet)
	fmt.Fprintf(w, "-debug:\t\t%v\t\t%v\n", *f.Debug, *a.Debug)
	fmt.Fprintf(w, "-yes:\t\t%v\t\t%v\n", *f.Yes, *a.Yes)
	fmt.Fprintf(w, "-version:\t\t%v\t\t%v\n", *f.Version, *a.Version)
	fmt.Fprintf(w, "-help:\t\t%v\t\t%v\n", *f.Help, *a.Help)
	fmt.Fprintf(w, "-exact:\t\t%v\t\t%v\n", *f.Exact, *a.Exact)
	fmt.Fprintf(w, "-name:\t\t%v\t\t%v\n", *f.Filename, *a.Filename)
	fmt.Fprintf(w, "-fast:\t\t%v\t\t%v\n", *f.Lookup, *a.Lookup)
	fmt.Fprintf(w, "-delete:\t\t%v\t\t%v\n", *f.Rm, na)
	fmt.Fprintf(w, "-delete+:\t\t%v\t\t%v\n", *f.RmPlus, na)
	fmt.Fprintf(w, "-sensen:\t\t%v\t\t%v\n", *f.RmPlus, na)
	w.Flush()
	return buf.String()
}

// DatabaseHelp creates the database command help.
func DatabaseHelp(w io.Writer) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, color.Primary.Sprint("Database commands:"))
	fmt.Fprintln(w, "  View information and run optional maintenance on the internal database.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Usage:")
	fmt.Fprintf(w, "    dupers %s\tdisplay statistics and bucket information\n", Database_)
	fmt.Fprintf(w, "    dupers %s\t%s\n", Backup_, "make a copy of the database to")
	fmt.Fprintf(w, "\t %s", color.Info.Sprint(cmd.Home()))
	fmt.Fprintln(w)
	fmt.Fprintf(w, "    dupers %s\t%s\n", Clean_, "compact and remove all items in the database that")
	fmt.Fprintln(w, "\t point to missing files")
	fmt.Fprintf(w, "    dupers %s <bucket>\t%s\n", LS_, "list the hashes and files in the bucket")
	fmt.Fprintf(w, "    dupers %s <bucket>\t%s\n", Up_, "add or update the bucket to the database")
	fmt.Fprintf(w, "    dupers %s <bucket>\t%s", UpPlus_, color.Danger.Sprint("(SLOW)"))
	fmt.Fprintln(w, " add or update the bucket using an archive")
	fmt.Fprintln(w, "\t scan the scan reads every file archived with known")
	fmt.Fprintln(w, "\t package formats")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "    dupers %s <bucket>\t%s\n", RM_, "remove the bucket from the database")
	fmt.Fprintf(w, "    dupers %s <bucket> <destination>\t%s\n", MV_, "move the bucket to a new directory path")
	fmt.Fprintf(w, "    dupers %s <bucket>\t%s\n", Export_, "export the bucket to a text file in")
	fmt.Fprintf(w, "\t %s", color.Info.Sprint(cmd.Home()))
	fmt.Fprintln(w)
	fmt.Fprintf(w, "    dupers %s <export file>\t%s\n", Import_, "import a bucket text file into the")
	fmt.Fprintln(w, "\t database")
}

// DupeHelp creates the dupe command help.
func DupeHelp(w io.Writer) {
	f := flag.Flag{}
	const danger = "(!)"
	fmt.Fprintln(w)
	fmt.Fprintln(w, color.Primary.Sprint("Dupe command:"))
	fmt.Fprintln(w, "  Scan for duplicate files, matching files that share the identical content.")
	fmt.Fprintln(w, "  The \"directory or file to check\" is never added to the database.")
	fmt.Fprint(w, "  The \"buckets to lookup\" are directories ")
	if runtime.GOOS == winOS {
		fmt.Fprint(w, "or drive letters ")
	}
	fmt.Fprintln(w, "that get added to the database for")
	fmt.Fprintln(w, "   quicker scans.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Usage:")
	fmt.Fprintln(w, "    dupers [options] dupe <directory or file to check> [buckets to lookup]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Options:")
	if flag.Lookup(cmd.Fast_) != nil {
		//fmt.Fprintln(w, "-----------------------------------------------------------------------------80|")
		f = *flag.Lookup(cmd.Fast_)
		fmt.Fprintf(w, "    -%s, -%s\t%s\n", f.Name[:1], f.Name, f.Usage)
		f = *flag.Lookup(cmd.Delete_)
		fmt.Fprintf(w, "        -%s\t%s ", f.Name, color.Danger.Sprint(danger))
		fmt.Fprintln(w, f.Usage)
		f = *flag.Lookup(cmd.DelPlus_)
		fmt.Fprintf(w, "        -%s\t%s ", f.Name, color.Danger.Sprint(danger))
		fmt.Fprintln(w, f.Usage)
		f = *flag.Lookup(cmd.Sensen_)
		fmt.Fprintf(w, "        -%s\t%s ", f.Name, color.Danger.Sprint(danger))
		fmt.Fprintln(w, f.Usage)
		fmt.Fprintln(w)
		fmt.Fprintln(w, color.Danger.Sprint("    ", danger), "this option is potentionally dangerous")
	}
	DupeExample(w)
}

// DupeExample creates the examples of the dupe command.
func DupeExample(w io.Writer) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Examples:")
	const a = "    # find identical copies of file.txt in the Downloads directory"
	fmt.Fprintln(w, color.Secondary.Sprint(a))
	if runtime.GOOS == winOS {
		dupeWindows(w)
		return
	}
	fmt.Fprintln(w, color.Info.Sprintf("    dupers dupe '%s' '%s'",
		filepath.Join(cmd.Home(), "file.txt"),
		filepath.Join(cmd.Home(), "Downloads")))
	const b = "    # search for files in Documents that also exist in /var/www"
	fmt.Fprintln(w, color.Secondary.Sprint(b))
	fmt.Fprintln(w, color.Info.Sprintf("    dupers dupe '%s' '%s'",
		filepath.Join(cmd.Home(), "Documents"), "/var/www"))
}

func dupeWindows(w io.Writer) {
	fmt.Fprintln(w, color.Info.Sprintf("    dupers dupe \"%s\" \"%s\"",
		filepath.Join(cmd.Home(), "file.txt"), filepath.Join(cmd.Home(), "Downloads")))
	const a = "    # search the database for files in Documents that also exist on drives D: and E:"
	fmt.Fprintln(w, color.Secondary.Sprint(a))
	fmt.Fprintln(w, color.Info.Sprintf("    dupers dupe \"%s\" %s %s",
		filepath.Join(cmd.Home(), "Documents"), "D:", "E:"))
}

func ProgramOpts(w io.Writer) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, color.Primary.Sprint("Options:"))
	fmt.Fprintln(w, "  Program options that can be used with any command.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Usage:")
	if flag.Lookup(cmd.Mono_) != nil {
		f := flag.Flag{}
		f = *flag.Lookup(cmd.Mono_)
		fmt.Fprintf(w, "    -%v, -%v\t%v\n", f.Name[:1], f.Name, f.Usage)
		f = *flag.Lookup(cmd.Quiet_)
		fmt.Fprintf(w, "    -%v, -%v\t%v\n", f.Name[:1], f.Name, f.Usage)
		f = *flag.Lookup(cmd.Yes_)
		fmt.Fprintf(w, "    -%v, -%v\t%v\n", f.Name[:1], f.Name, f.Usage)
		f = *flag.Lookup(cmd.Debug_)
		fmt.Fprintf(w, "    -%v, -%v\t%v\n", f.Name[:1], f.Name, f.Usage)
		f = *flag.Lookup(cmd.Version_)
		fmt.Fprintf(w, "    -%v, -%v\t%v\n", f.Name[:1], f.Name, f.Usage)
	}
	fmt.Fprintf(w, "    -h, %s\tshow this list of options\n", "-help")
}

// Search creates the search command help.
func SearchHelp(w io.Writer) {
	f := flag.Flag{}
	fmt.Fprintln(w)
	fmt.Fprintln(w, color.Primary.Sprint("Search command:"))
	fmt.Fprintln(w, "  Lookup a file or a directory name in the database.")
	fmt.Fprintln(w, "  The <search expression> can be a partial or complete, file or directory name.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Usage:")
	fmt.Fprintln(w, "    dupers [options] search <search expression> [optional, buckets to search]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Options:")
	if flag.Lookup(cmd.Exact_) != nil {
		f = *flag.Lookup(cmd.Exact_)
		fmt.Fprintf(w, "    -%v, -%v\t\t%v\n", f.Name[:1], f.Name, f.Usage)
		f = *flag.Lookup(cmd.Name_)
		fmt.Fprintf(w, "    -%v, -%v\t\t%v\n", f.Name[:1], f.Name, f.Usage)
	}
	SearchExample(w)
}

// SearchExample creates the examples of the search command.
func SearchExample(w io.Writer) {
	fmt.Fprintln(w, "\n  Examples:")
	const a = "    # search for the expression foo in your home directory"
	fmt.Fprintln(w, color.Secondary.Sprint(a))
	if runtime.GOOS == winOS {
		fmt.Fprint(w, "    "+color.Info.Sprintf("dupers search \"foo\" \"%s\"", cmd.Home()))
		fmt.Fprint(w, color.Secondary.Sprint("\n    # search for filenames containing .zip\n"))
		fmt.Fprint(w, "    "+color.Info.Sprint("dupers -name search \".zip\""))
		fmt.Fprintln(w)
		return
	}
	fmt.Fprint(w, "    "+color.Info.Sprintf("dupers search 'foo' '%s'", cmd.Home()))
	fmt.Fprint(w, color.Secondary.Sprint("\n    # search for filenames containing .zip\n"))
	fmt.Fprint(w, "    "+color.Info.Sprint("dupers -name search '.zip'"))
	fmt.Fprintln(w)
}
