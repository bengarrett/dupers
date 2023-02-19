// © Ben Garrett https://github.com/bengarrett/dupers
package duplicate

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/bengarrett/dupers/internal/cmd"
	"github.com/bengarrett/dupers/internal/out"
	"github.com/bengarrett/dupers/pkg/database"
	"github.com/bengarrett/dupers/pkg/dupe"
	"github.com/gookit/color"
	bolt "go.etcd.io/bbolt"
)

var (
	ErrFast   = errors.New("fast flag cannot be used in this situation")
	ErrNoArgs = errors.New("request is missing arguments")
	ErrSearch = errors.New("search request needs an expression")
)

const winOS = "windows"

// Cleanup runs the cleanup commands when the appropriate flags are set.
func Cleanup(c *dupe.Config, f *cmd.Flags) error {
	if c == nil {
		return dupe.ErrNilConfig
	}
	if f == nil {
		return cmd.ErrNilFlag
	}
	if f.Sensen == nil {
		return fmt.Errorf("%w: sensen", cmd.ErrNilFlag)
	}
	if f.Rm == nil {
		return fmt.Errorf("%w: rm", cmd.ErrNilFlag)
	}
	if f.RmPlus == nil {
		return fmt.Errorf("%w: rmplus", cmd.ErrNilFlag)
	}
	if f.Yes == nil {
		return fmt.Errorf("%w: yes", cmd.ErrNilFlag)
	}
	if !*f.Sensen && !*f.Rm && !*f.RmPlus {
		return nil
	}
	w := os.Stdout
	if *f.Sensen {
		c.DPrint("remove all non unique Windows and MS-DOS files.")
		s, err := c.Remove()
		if err != nil {
			return err
		}
		fmt.Fprint(w, s)
		removes, err := c.Removes(*f.Yes)
		if err != nil {
			return err
		}
		for _, name := range removes {
			var err = errors.New("could not remove file")
			dupe.PrintRM(name, err)
		}
		fmt.Fprint(w, s)
		fmt.Fprint(w, c.Clean())
		return nil
	}
	if *f.Rm || *f.RmPlus {
		c.DPrint("remove duplicate files.")
		s, err := c.Remove()
		if err != nil {
			return err
		}
		fmt.Fprint(w, s)
		if *f.RmPlus {
			c.DPrint("remove empty directories.")
			fmt.Fprint(w, c.Clean())
		}
	}
	return nil
}

// CmdErr parses the arguments of the dupe command.
func CmdErr(args, buckets, minArgs int, test bool) {
	w := os.Stdout
	if args < minArgs {
		out.StderrCR(ErrNoArgs)
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

// WalkScanSave both cleans and then updates the buckets with file system changes.
func WalkScanSave(db *bolt.DB, c *dupe.Config, f *cmd.Flags) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	if c == nil {
		return dupe.ErrNilConfig
	}
	if f == nil || f.Lookup == nil {
		return fmt.Errorf("%w: lookup", cmd.ErrNilFlag)
	}
	c.DPrint("dupe lookup.")

	var errs error

	// normalise bucket names

	// TODO: CHECK EXISTANCE
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

	if !*f.Lookup && len(buckets) > 0 {
		c.DPrint("non-fast mode, database cleanup.")
		if err := database.Clean(db, c.Quiet, c.Debug, buckets...); err != nil {
			out.StderrCR(err)
		}
	}
	if *f.Lookup {
		if err := Lookup(db, c); err != nil {
			return nil
		}
	}
	c.DPrint("walk the buckets.")
	return c.WalkDirs(db)
}

func Lookup(db *bolt.DB, c *dupe.Config) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	if c == nil {
		return dupe.ErrNilConfig
	}
	c.DPrint("read the hash values in the buckets.")
	fastFlagErr := false
	for _, bucket := range c.All() {
		if i, err := c.SetCompares(db, bucket); err != nil {
			out.StderrCR(err)
		} else if i > 0 {
			continue
		}
		fastFlagErr = true
		fmt.Fprintln(os.Stderr, "The -fast flag cannot be used for this dupe query")
	}
	if fastFlagErr {
		return ErrFast
	}
	return nil
}
