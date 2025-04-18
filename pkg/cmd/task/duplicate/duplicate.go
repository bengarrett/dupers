// © Ben Garrett https://github.com/bengarrett/dupers
package duplicate

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/bengarrett/dupers/internal/printer"
	"github.com/bengarrett/dupers/pkg/cmd"
	"github.com/bengarrett/dupers/pkg/database"
	"github.com/bengarrett/dupers/pkg/dupe"
	"github.com/gookit/color"
	bolt "go.etcd.io/bbolt"
	boltErr "go.etcd.io/bbolt/errors"
)

var (
	ErrFast   = errors.New("fast flag cannot be used in this situation")
	ErrNoArgs = errors.New("request is missing arguments")
	ErrNoRM   = errors.New("could not remove file")
	ErrSearch = errors.New("search request needs an expression")
)

const winOS = "windows"

// Cleanup runs the cleanup commands when the appropriate flags are set.
func Cleanup(c *dupe.Config, f *cmd.Flags) error {
	if c == nil {
		return dupe.ErrNilConfig
	}
	if err := checkF(f); err != nil {
		return err
	}
	w := os.Stdout
	if *f.Rm || *f.RmPlus || *f.Sensen {
		s, err := c.Remove()
		if err != nil {
			return err
		}
		fmt.Fprint(w, s)
	}
	if *f.Sensen {
		removes, err := c.Removes()
		if err != nil {
			return err
		}
		for _, name := range removes {
			dupe.PrintRM(name, ErrNoRM)
		}
	}
	if *f.RmPlus || *f.Sensen {
		if err := c.Clean(w); err != nil {
			return err
		}
	}
	return nil
}

func checkF(f *cmd.Flags) error {
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
	return nil
}

// Check parses the arguments of the dupe command.
func Check(args, buckets, minArgs int) {
	w := os.Stdout
	if args < minArgs {
		printer.StderrCR(ErrNoArgs)
		fmt.Fprintln(w, "\nThe dupe command requires a directory or file to check.")
		if runtime.GOOS == winOS {
			fmt.Fprintln(w, "The optional bucket can be one or more directories or drive letters.")
		} else {
			fmt.Fprintln(w, "The optional bucket can be one or more directory paths.")
		}
		printer.Example("\ndupers dupe <directory or file to check> [buckets to lookup]")
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
			printer.Example(s)
		}
	}
}

// WalkScanSave both cleans and then updates the buckets with file system changes.
func WalkScanSave(db *bolt.DB, c *dupe.Config, f *cmd.Flags) error {
	if db == nil {
		return boltErr.ErrDatabaseNotOpen
	}
	if c == nil {
		return dupe.ErrNilConfig
	}
	if f == nil || f.Lookup == nil {
		return fmt.Errorf("%w: lookup", cmd.ErrNilFlag)
	}
	c.Debugger("dupe lookup.")

	if err := normalise(db, c); err != nil {
		return err
	}

	buckets := make([]string, 0, len(c.Buckets))
	for _, b := range c.Buckets {
		buckets = append(buckets, string(b))
	}

	if !*f.Lookup && len(buckets) > 0 {
		c.Debugger("non-fast mode, database cleanup.")
		if err := database.Clean(db, c.Quiet, c.Debug, buckets...); err != nil {
			printer.StderrCR(err)
		}
	}
	if *f.Lookup {
		if err := Lookup(db, c); err != nil {
			return nil
		}
	}
	c.Debugger("walk the buckets.")
	return c.WalkDirs(db)
}

// normalise the names of the buckets.
func normalise(db *bolt.DB, c *dupe.Config) error {
	var errs error
	for i, b := range c.Buckets {
		abs, err := database.Abs(string(b))
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("x %w: %s", err, b))
			c.Buckets[i] = ""

			continue
		}
		if err := database.Exist(db, abs); err != nil {
			if errors.Is(err, bolt.ErrBucketNotFound) {
				s := "Scan and add this new directory to the database:\n" +
					color.Warn.Sprintf(" %s\n\n", abs) +
					"This could take awhile with large numbers of files"
				if printer.AskYN(s, c.Yes, printer.Yes) {
					// continue automatically adds the bucket to the database
					continue
				}
			}
			errs = errors.Join(errs, fmt.Errorf("y %w: %s", err, b))
		}
		c.Buckets[i] = dupe.Bucket(abs)
	}
	if errs != nil {
		return errs
	}
	return nil
}

func Lookup(db *bolt.DB, c *dupe.Config) error {
	if db == nil {
		return boltErr.ErrDatabaseNotOpen
	}
	if c == nil {
		return dupe.ErrNilConfig
	}
	c.Debugger("read the hash values in the buckets.")
	fastFlagErr := false
	for _, bucket := range c.Buckets {
		if i, err := c.SetCompares(db, bucket); err != nil {
			printer.StderrCR(err)
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
