// © Ben Garrett https://github.com/bengarrett/dupers
package duplicate

import (
	"errors"
	"flag"
	"fmt"
	"os"
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
	w := os.Stdout
	if *f.Sensen {
		c.DPrint("remove all non unique Windows and MS-DOS files.")
		fmt.Fprint(w, c.Remove())
		fmt.Fprint(w, c.Removes(*f.Yes))
		fmt.Fprint(w, c.Clean())
		return
	}
	if *f.Rm || *f.RmPlus {
		c.DPrint("remove duplicate files.")
		fmt.Fprint(w, c.Remove())
		if *f.RmPlus {
			c.DPrint("remove empty directories.")
			fmt.Fprint(w, c.Clean())
		}
	}
}

// CmdErr parses the arguments of the dupe command.
func CmdErr(args, buckets, minArgs int, test bool) {
	w := os.Stdout
	if args < minArgs {
		out.ErrCont(ErrNoArgs)
		fmt.Fprintln(w, "\nThe dupe command requires a directory or file to check.")
		if runtime.GOOS == winOS {
			fmt.Fprintln(w, "The optional bucket can be one or more directories or drive letters.")
		} else {
			fmt.Fprintln(w, "The optional bucket can be one or more directory paths.")
		}
		out.Example("\ndupers dupe <directory or file to check> [buckets to lookup]")
	}
	if buckets == 0 && args == minArgs {
		fmt.Fprintln(w, color.Warn.Sprint("The database is empty.\n"))
		if runtime.GOOS == winOS {
			fmt.Fprintln(w, "This dupe request requires at least one directory or drive letter to lookup.")
		} else {
			fmt.Fprintln(w, "This dupe request requires at least one directory to lookup.")
		}
		fmt.Fprintln(w, "These lookup directories will be stored to the database as buckets.")
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
func Lookup(c *dupe.Config, f *cmd.Flags) error {
	c.DPrint("dupe lookup.")

	var errs error

	db := c.Parser.DB
	if db == nil {
		db, err := database.OpenRead()
		if err != nil {
			return err
		}
		defer db.Close()
	}

	// normalise bucket names

	// CHECK EXISTANCE
	for i, b := range c.All() {
		abs, err := database.Abs(string(b))
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("%w: %s", err, b))
			c.All()[i] = ""

			continue
		}
		if err := database.Exist(db, abs); err != nil {
			errs = errors.Join(errs, fmt.Errorf("%w: %s", err, b))
		}
		c.All()[i] = dupe.Bucket(abs)
	}
	if errs != nil {
		return errs
	}

	buckets := make([]string, 0, len(c.All()))
	for _, b := range c.All() {
		buckets = append(buckets, string(b))
	}

	fmt.Println(" >>> ", buckets)

	if !*f.Lookup && len(buckets) > 0 {
		c.DPrint("non-fast mode, database cleanup.")
		if err := database.Clean(c.Parser.DB, c.Quiet, c.Debug, buckets...); err != nil {
			out.ErrCont(err)
		}
	}
	if *f.Lookup {
		if err := lookup(c); err != nil {
			return nil
		}
	}
	c.DPrint("walk the buckets.")
	c.WalkDirs()
	return nil
}

func lookup(c *dupe.Config) error {
	c.DPrint("read the hash values in the buckets.")
	fastErr := false
	for _, bucket := range c.All() {
		if i, err := c.SetCompares(bucket); err != nil {
			fmt.Println("---------------> OOPS")
			out.ErrCont(err)
		} else if i > 0 {
			continue
		}
		fastErr = true
		fmt.Fprintln(os.Stderr, "The -fast flag cannot be used for this dupe query")
	}
	if !fastErr {
		return ErrFast
	}
	return nil
}
