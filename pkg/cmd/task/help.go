package task

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"slices"
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
	printl(w, color.Primary.Sprint("DATABASE commands:"))
	printl(w, "  View information and run optional maintenance on the internal database.")
	printl(w)
	printl(w, "  Usage:")
	printf(w, "    dupers %s\t%s\n", Database_, "display statistics and bucket information")
	printf(w, "    dupers %s\t%s\n", Backup_, "make a copy of the database")
	printf(w, "    dupers %s\t%s\n", Clean_, "compact and remove items pointing to missing files")
	printf(w, "    dupers %s <bucket>\t%s\n", LS_, "list the hashes and files in the bucket")
	printf(w, "    dupers %s <bucket>\t%s\n", Up_, "add or update the bucket to the database")
	printf(w, "    dupers %s <bucket>\t%s\n", UpPlus_, color.Danger.Sprint("(SLOW) add bucket using archives scan"))
	printf(w, "    dupers %s <bucket>\t%s\n", RM_, "remove the bucket from the database")
	printf(w, "    dupers %s <bucket> <dest>\t%s\n", MV_, "move the bucket to a new directory path")
	printf(w, "    dupers %s <bucket>\t%s\n", Export_, "export the bucket to a text file")
	printf(w, "    dupers %s <export file>\t%s\n", Import_, "import a bucket text file into the database")
}

// DupeHelp creates the dupe command help.
func DupeHelp(w io.Writer) {
	const danger = "(!)"
	printl(w, color.Primary.Sprint("DUPE command:"))
	printl(w, "  Scan for duplicate files with identical content. The file or \n"+
		"  directory being checked is never added to the database, while\n  lookup buckets are directories.")
	printl(w)
	printl(w, "  Usage:")
	printl(w, "    dupers [options] dupe <directory or file to check> [buckets to lookup]")
	printl(w)
	printl(w, "  Options:")
	if flag.Lookup(cmd.Fast_) != nil {
		var f *flag.Flag
		f = flag.Lookup(cmd.Fast_)
		if len(f.Name) > 1 {
			printf(w, "    -%s, -%s\t%s\n", f.Name[:1], f.Name, f.Usage)
		}
		f = flag.Lookup(cmd.Delete_)
		if f != nil {
			printf(w, "        -%s\t%s ", f.Name, color.Danger.Sprint(danger))
			printl(w, f.Usage)
		}
		f = flag.Lookup(cmd.DelPlus_)
		if f != nil {
			printf(w, "        -%s\t%s ", f.Name, color.Danger.Sprint(danger))
			printl(w, f.Usage)
		}
		f = flag.Lookup(cmd.Sensen_)
		if f != nil {
			printf(w, "        -%s\t%s ", f.Name, color.Danger.Sprint(danger))
			printl(w, f.Usage)
		}
	}
	DupeExample(w)
}

// QuickStartHelp creates a quick start guide for new users.
func QuickStartHelp(w io.Writer) {
	printl(w)
	printl(w, color.Primary.Sprint("Quick Start:"))
	printl(w, "  Get started with Dupers in just a few commands.")
	printl(w)
	printl(w, "  1. Add your main directories to the database (buckets):")
	printl(w, color.Info.Sprintf("     dupers up ~/Documents"))
	printl(w, color.Info.Sprintf("     dupers up ~/Downloads"))
	printl(w, color.Info.Sprintf("     dupers up /path/to/your/files"))
	printl(w)
	printl(w, "  2. Find duplicate files:")
	printl(w, color.Info.Sprintf("     dupers dupe ~/Pictures ~/Documents"))
	printl(w)
	printl(w, "  3. Search for files by name:")
	printl(w, color.Info.Sprintf("     dupers search 'project'"))
	printl(w)
	printl(w, "  4. View database information:")
	printl(w, color.Info.Sprintf("     dupers database"))
	printl(w)
}

// DupeExample creates the examples of the dupe command.
func DupeExample(w io.Writer) {
	printl(w, "  Examples:")
	const a = "  1. Find identical copies of file.txt in the Downloads directory"
	printl(w, color.Secondary.Sprint(a))
	if runtime.GOOS == winOS {
		dupeWindows(w)
		return
	}
	printl(w, color.Info.Sprintf("     dupers dupe '%s' '%s'",
		filepath.Join(cmd.Home(), "file.txt"),
		filepath.Join(cmd.Home(), "Downloads")))
	const b = "  2. Search for files in Documents that also exist in /var/www"
	printl(w, color.Secondary.Sprint(b))
	printl(w, color.Info.Sprintf("     dupers dupe '%s' '%s'",
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
	printl(w, color.Primary.Sprint("OPTION flags:"))
	printl(w, "  Program options that can be used with any command.")
	printl(w)
	printl(w, "  Usage:")
	if flag.Lookup(cmd.Mono_) != nil {
		lcmd := []string{cmd.Mono_, cmd.Quiet_, cmd.Yes_, cmd.Debug_, cmd.Version_}
		var f *flag.Flag
		for name := range slices.Values(lcmd) {
			f = flag.Lookup(name)
			if f == nil {
				continue
			}
			printf(w, "    -%v, -%v\t%v\n", f.Name[:1], f.Name, f.Usage)
		}
	}
	printf(w, "    -h, %s\tshow this list of options\n", "-help")
}

// SearchHelp creates the search command help.
func SearchHelp(w io.Writer) {
	printl(w)
	printl(w, color.Primary.Sprint("SEARCH command:"))
	printl(w, "  Lookup a file or a directory name in the database.")
	printl(w, "  The <search expression> can be a partial or complete, file or directory name.")
	printl(w)
	printl(w, "  Usage:")
	printl(w, "    dupers [options] search <search expression> [optional, buckets to search]")
	printl(w)
	printl(w, "  Options:")
	if flag.Lookup(cmd.Exact_) != nil {
		var f *flag.Flag
		f = flag.Lookup(cmd.Exact_)
		if f != nil {
			printf(w, "    -%v, -%v\t\t%v\n", f.Name[:1], f.Name, f.Usage)
		}
		f = flag.Lookup(cmd.Name_)
		if f != nil {
			printf(w, "    -%v, -%v\t\t%v\n", f.Name[:1], f.Name, f.Usage)
		}
	}
	SearchExample(w)
}

// SearchExample creates the examples of the search command.
func SearchExample(w io.Writer) {
	printl(w, "\n  Examples:")
	const a = "  1. Search for the expression foo in your home directory"
	printl(w, color.Secondary.Sprint(a))
	if runtime.GOOS == winOS {
		printr(w, pad4+color.Info.Sprintf(" dupers search \"foo\" \"%s\"", cmd.Home()))
		printr(w, color.Secondary.Sprint("\n  2. Search for filenames containing .zip\n"))
		printr(w, pad4+color.Info.Sprint(" dupers -name search \".zip\""))
		printl(w)
		return
	}
	printr(w, pad4+color.Info.Sprintf(" dupers search 'foo' '%s'", cmd.Home()))
	printr(w, color.Secondary.Sprint("\n  2. Search for filenames containing .zip\n"))
	printr(w, pad4+color.Info.Sprint(" dupers -name search '.zip'"))
	printl(w)
}
