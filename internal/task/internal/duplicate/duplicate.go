// © Ben Garrett https://github.com/bengarrett/dupers
package duplicate

import (
	"errors"
	"flag"
	"fmt"
	"runtime"

	"github.com/bengarrett/dupers/database"
	"github.com/bengarrett/dupers/dupe"
	"github.com/bengarrett/dupers/internal/cmd"
	"github.com/bengarrett/dupers/internal/out"
	"github.com/gookit/color"
)

var (
	ErrFast   = errors.New("fast flag cannot be used in this situation")
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
			out.PBug("remove all non unique Windows and MS-DOS files.")
		}
		fmt.Print(c.Remove())
		fmt.Print(c.Removes())
		fmt.Print(c.Clean())
		return
	}
	if *f.Rm || *f.RmPlus {
		if c.Debug {
			out.PBug("remove duplicate files.")
		}
		fmt.Print(c.Remove())
		if *f.RmPlus {
			if c.Debug {
				out.PBug("remove empty directories.")
			}
			fmt.Print(c.Clean())
		}
	}
}

// CmdErr parses the arguments of the dupe command.
func CmdErr(args, buckets, minArgs int, test bool) {
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
		if len(flag.Args()) > 0 {
			s := fmt.Sprintf("\ndupers dupe %s <one or more directories>\n", flag.Args()[1])
			out.Example(s)
		}
	}
	if !test {
		out.ErrFatal(nil)
	}
}

// Lookup both cleans and then updates the buckets with file system changes.
func Lookup(c *dupe.Config, f *cmd.Flags) {
	if c.Debug {
		out.PBug("dupe lookup.")
	}
	// normalise bucket names
	for i, b := range c.All() {
		abs, err := database.Abs(string(b))
		if err != nil {
			out.ErrCont(err)
			c.All()[i] = ""

			continue
		}
		c.All()[i] = dupe.Bucket(abs)
	}
	buckets := make([]string, 0, len(c.All()))
	for _, b := range c.All() {
		buckets = append(buckets, string(b))
	}
	if !*f.Lookup && len(buckets) > 0 {
		if c.Debug {
			out.PBug("non-fast mode, database cleanup.")
		}
		if err := database.Clean(c.Quiet, c.Debug, buckets...); err != nil {
			out.ErrCont(err)
		}
	}
	if *f.Lookup {
		if err := lookup(c); err != nil {
			return
		}
	}
	if c.Debug {
		out.PBug("walk the buckets.")
	}
	c.WalkDirs()
}

func lookup(c *dupe.Config) error {
	if c.Debug {
		out.PBug("read the hash values in the buckets.")
	}
	fastErr := false
	for _, b := range c.All() {
		if i, err := c.SetCompares(b); err != nil {
			out.ErrCont(err)
		} else if i > 0 {
			continue
		}
		fastErr = true
		fmt.Println("The -fast flag cannot be used for this dupe query")
	}
	if !fastErr {
		return ErrFast
	}
	return nil
}
