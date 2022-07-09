// © Ben Garrett https://github.com/bengarrett/dupers
package duplicate

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"runtime"

	dupe "github.com/bengarrett/dupers"
	"github.com/bengarrett/dupers/internal/cmd"
	"github.com/bengarrett/dupers/internal/out"
	"github.com/gookit/color"
)

var (
	ErrNoArgs = errors.New("request is missing arguments")
	ErrSearch = errors.New("search request needs an expression")
)

const (
	winOS = "windows"
)

// Cleanup runs the cleanup commands when the appropriate flags are set.
func Cleanup(c *dupe.Config, f *cmd.Flags) {
	if *f.Sensen {
		if c.Debug {
			out.DebugLn("remove all non unique Windows and MS-DOS files.")
		}
		fmt.Print(c.Remove())
		fmt.Print(c.Removes())
		fmt.Print(c.Clean())
		return
	}
	if *f.Rm || *f.RmPlus {
		if c.Debug {
			out.DebugLn("remove duplicate files.")
		}
		fmt.Print(c.Remove())
		if *f.RmPlus {
			if c.Debug {
				out.DebugLn("remove empty directories.")
			}
			fmt.Print(c.Clean())
		}
	}
}

// CmdErr parses the arguments of the dupe command.
func CmdErr(args, buckets, minArgs int) string {
	var b bytes.Buffer
	if args < minArgs {
		out.ErrCont(ErrNoArgs)
		fmt.Fprintln(&b, "\nThe dupe command requires a directory or file to check.")
		if runtime.GOOS == winOS {
			fmt.Fprintln(&b, "The optional bucket can be one or more directories or drive letters.")
		} else {
			fmt.Fprintln(&b, "The optional bucket can be one or more directory paths.")
		}
		fmt.Fprintln(&b, out.Example("\ndupers dupe <directory or file to check> [buckets to lookup]"))
	}
	if buckets == 0 && args == minArgs {
		fmt.Fprintln(&b, color.Warn.Sprint("The database is empty.\n"))
		if runtime.GOOS == winOS {
			fmt.Fprintln(&b, "This dupe request requires at least one directory or drive letter to lookup.")
		} else {
			fmt.Fprintln(&b, "This dupe request requires at least one directory to lookup.")
		}
		fmt.Fprintln(&b, "These lookup directories will be stored to the database as buckets.")
		if len(flag.Args()) > 0 {
			s := fmt.Sprintf("\ndupers dupe %s <one or more directories>\n\n", flag.Args()[1])
			fmt.Fprint(&b, s)
		}
	}
	return b.String()
}
