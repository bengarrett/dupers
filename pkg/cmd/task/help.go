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

	"github.com/bengarrett/dupers/pkg/cmd"
	"github.com/gookit/color"
)

const pad4 = "    "

func printr(w io.Writer, a ...any) {
	_, _ = fmt.Fprint(w, a...)
}

func printl(w io.Writer, a ...any) {
	_, _ = fmt.Fprintln(w, a...)
}

func printf(w io.Writer, format string, a ...any) {
	_, _ = fmt.Fprintf(w, format, a...)
}

func Debug(a *cmd.Aliases, f *cmd.Flags) (string, error) {
	if a == nil {
		return "", cmd.ErrNilAlias
	}
	if f == nil {
		return "", cmd.ErrNilFlag
	}
	const na = "n/a"
	buf := new(bytes.Buffer)
	w := tabwriter.NewWriter(buf, 0, 0, 1, ' ', tabwriter.AlignRight)
	printl(w, "Dupers arguments debug:")
	printl(w)
	printl(w, "\t\tToggle\t\tAlias")
	printf(w, "-mono:\t\t%v\t\t%v\n", *f.Mono, *a.Mono)
	printf(w, "-quiet:\t\t%v\t\t%v\n", *f.Quiet, *a.Quiet)
	printf(w, "-debug:\t\t%v\t\t%v\n", *f.Debug, *a.Debug)
	printf(w, "-yes:\t\t%v\t\t%v\n", *f.Yes, *a.Yes)
	printf(w, "-version:\t\t%v\t\t%v\n", *f.Version, *a.Version)
	printf(w, "-help:\t\t%v\t\t%v\n", *f.Help, *a.Help)
	printf(w, "-exact:\t\t%v\t\t%v\n", *f.Exact, *a.Exact)
	printf(w, "-name:\t\t%v\t\t%v\n", *f.Filename, *a.Filename)
	printf(w, "-fast:\t\t%v\t\t%v\n", *f.Lookup, *a.Lookup)
	printf(w, "-delete:\t\t%v\t\t%v\n", *f.Rm, na)
	printf(w, "-delete+:\t\t%v\t\t%v\n", *f.RmPlus, na)
	printf(w, "-sensen:\t\t%v\t\t%v\n", *f.RmPlus, na)
	if err := w.Flush(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// DatabaseHelp creates the database command help.
func DatabaseHelp(w io.Writer) {
	printl(w)
	printl(w, color.Primary.Sprint("Database commands:"))
	printl(w, "  View information and run optional maintenance on the internal database.")
	printl(w)
	printl(w, "  Usage:")
	printf(w, "    dupers %s\tdisplay statistics and bucket information\n", Database_)
	printf(w, "    dupers %s\t%s\n", Backup_, "make a copy of the database to")
	printf(w, "\t %s", color.Info.Sprint(cmd.Home()))
	printl(w)
	printf(w, "    dupers %s\t%s\n", Clean_, "compact and remove all items in the database that")
	printl(w, "\t point to missing files")
	printf(w, "    dupers %s <bucket>\t%s\n", LS_, "list the hashes and files in the bucket")
	printf(w, "    dupers %s <bucket>\t%s\n", Up_, "add or update the bucket to the database")
	printf(w, "    dupers %s <bucket>\t%s", UpPlus_, color.Danger.Sprint("(SLOW)"))
	printl(w, " add or update the bucket using an archive")
	printl(w, "\t scan the scan reads every file archived with known")
	printl(w, "\t package formats")
	printl(w)
	printf(w, "    dupers %s <bucket>\t%s\n", RM_, "remove the bucket from the database")
	printf(w, "    dupers %s <bucket> <destination>\t%s\n", MV_, "move the bucket to a new directory path")
	printf(w, "    dupers %s <bucket>\t%s\n", Export_, "export the bucket to a text file in")
	printf(w, "\t %s", color.Info.Sprint(cmd.Home()))
	printl(w)
	printf(w, "    dupers %s <export file>\t%s\n", Import_, "import a bucket text file into the")
	printl(w, "\t database")
}

// DupeHelp creates the dupe command help.
func DupeHelp(w io.Writer) {
	var f flag.Flag
	const danger = "(!)"
	printl(w)
	printl(w, color.Primary.Sprint("Dupe command:"))
	printl(w, "  Scan for duplicate files, matching files that share the identical content.")
	printl(w, "  The \"directory or file to check\" is never added to the database.")
	printr(w, "  The \"buckets to lookup\" are directories ")
	if runtime.GOOS == winOS {
		printr(w, "or drive letters ")
	}
	printl(w, "that get added to the database for")
	printl(w, "   quicker scans.")
	printl(w)
	printl(w, "  Usage:")
	printl(w, "    dupers [options] dupe <directory or file to check> [buckets to lookup]")
	printl(w)
	printl(w, "  Options:")
	if flag.Lookup(cmd.Fast_) != nil {
		// fmt.Fprintln(w, "-----------------------------------------------------------------------------80|")
		f = *flag.Lookup(cmd.Fast_)
		printf(w, "    -%s, -%s\t%s\n", f.Name[:1], f.Name, f.Usage)
		f = *flag.Lookup(cmd.Delete_)
		printf(w, "        -%s\t%s ", f.Name, color.Danger.Sprint(danger))
		printl(w, f.Usage)
		f = *flag.Lookup(cmd.DelPlus_)
		printf(w, "        -%s\t%s ", f.Name, color.Danger.Sprint(danger))
		printl(w, f.Usage)
		f = *flag.Lookup(cmd.Sensen_)
		printf(w, "        -%s\t%s ", f.Name, color.Danger.Sprint(danger))
		printl(w, f.Usage)
		printl(w)
		printl(w, color.Danger.Sprint(pad4, danger), "this option is potentionally dangerous")
	}
	DupeExample(w)
}

// DupeExample creates the examples of the dupe command.
func DupeExample(w io.Writer) {
	printl(w)
	printl(w, "  Examples:")
	const a = "    # find identical copies of file.txt in the Downloads directory"
	printl(w, color.Secondary.Sprint(a))
	if runtime.GOOS == winOS {
		dupeWindows(w)
		return
	}
	printl(w, color.Info.Sprintf("    dupers dupe '%s' '%s'",
		filepath.Join(cmd.Home(), "file.txt"),
		filepath.Join(cmd.Home(), "Downloads")))
	const b = "    # search for files in Documents that also exist in /var/www"
	printl(w, color.Secondary.Sprint(b))
	printl(w, color.Info.Sprintf("    dupers dupe '%s' '%s'",
		filepath.Join(cmd.Home(), "Documents"), "/var/www"))
}

func dupeWindows(w io.Writer) {
	printl(w, color.Info.Sprintf("    dupers dupe \"%s\" \"%s\"",
		filepath.Join(cmd.Home(), "file.txt"), filepath.Join(cmd.Home(), "Downloads")))
	const a = "    # search the database for files in Documents that also exist on drives D: and E:"
	printl(w, color.Secondary.Sprint(a))
	printl(w, color.Info.Sprintf("    dupers dupe \"%s\" %s %s",
		filepath.Join(cmd.Home(), "Documents"), "D:", "E:"))
}

func ProgramOpts(w io.Writer) {
	printl(w)
	printl(w, color.Primary.Sprint("Options:"))
	printl(w, "  Program options that can be used with any command.")
	printl(w)
	printl(w, "  Usage:")
	if flag.Lookup(cmd.Mono_) != nil {
		var f flag.Flag
		f = *flag.Lookup(cmd.Mono_)
		printf(w, "    -%v, -%v\t%v\n", f.Name[:1], f.Name, f.Usage)
		f = *flag.Lookup(cmd.Quiet_)
		printf(w, "    -%v, -%v\t%v\n", f.Name[:1], f.Name, f.Usage)
		f = *flag.Lookup(cmd.Yes_)
		printf(w, "    -%v, -%v\t%v\n", f.Name[:1], f.Name, f.Usage)
		f = *flag.Lookup(cmd.Debug_)
		printf(w, "    -%v, -%v\t%v\n", f.Name[:1], f.Name, f.Usage)
		f = *flag.Lookup(cmd.Version_)
		printf(w, "    -%v, -%v\t%v\n", f.Name[:1], f.Name, f.Usage)
	}
	printf(w, "    -h, %s\tshow this list of options\n", "-help")
}

// Search creates the search command help.
func SearchHelp(w io.Writer) {
	var f flag.Flag
	printl(w)
	printl(w, color.Primary.Sprint("Search command:"))
	printl(w, "  Lookup a file or a directory name in the database.")
	printl(w, "  The <search expression> can be a partial or complete, file or directory name.")
	printl(w)
	printl(w, "  Usage:")
	printl(w, "    dupers [options] search <search expression> [optional, buckets to search]")
	printl(w)
	printl(w, "  Options:")
	if flag.Lookup(cmd.Exact_) != nil {
		f = *flag.Lookup(cmd.Exact_)
		printf(w, "    -%v, -%v\t\t%v\n", f.Name[:1], f.Name, f.Usage)
		f = *flag.Lookup(cmd.Name_)
		printf(w, "    -%v, -%v\t\t%v\n", f.Name[:1], f.Name, f.Usage)
	}
	SearchExample(w)
}

// SearchExample creates the examples of the search command.
func SearchExample(w io.Writer) {
	printl(w, "\n  Examples:")
	const a = "    # search for the expression foo in your home directory"
	printl(w, color.Secondary.Sprint(a))
	if runtime.GOOS == winOS {
		printr(w, pad4+color.Info.Sprintf("dupers search \"foo\" \"%s\"", cmd.Home()))
		printr(w, color.Secondary.Sprint("\n    # search for filenames containing .zip\n"))
		printr(w, pad4+color.Info.Sprint("dupers -name search \".zip\""))
		printl(w)
		return
	}
	printr(w, pad4+color.Info.Sprintf("dupers search 'foo' '%s'", cmd.Home()))
	printr(w, color.Secondary.Sprint("\n    # search for filenames containing .zip\n"))
	printr(w, pad4+color.Info.Sprint("dupers -name search '.zip'"))
	printl(w)
}
