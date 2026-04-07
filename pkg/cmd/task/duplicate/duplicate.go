// © Ben Garrett https://github.com/bengarrett/dupers

// Package duplicate provides duplicate file detection functionality.
package duplicate

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/bengarrett/dupers/internal/printer"
	"github.com/bengarrett/dupers/pkg/cmd"
	"github.com/bengarrett/dupers/pkg/database"
	"github.com/bengarrett/dupers/pkg/dupe"
	"github.com/gookit/color"
	bolt "go.etcd.io/bbolt"
	bberr "go.etcd.io/bbolt/errors"
)

var (
	ErrFast   = errors.New("fast flag cannot be used in this situation")
	ErrNoArgs = errors.New("request is missing arguments")
	ErrNoRM   = errors.New("could not remove file")
	ErrSearch = errors.New("search request needs an expression")
)

const winOS = "windows"

func printl(w io.Writer, a ...any) {
	_, _ = fmt.Fprintln(w, a...)
}

// Cleanup runs the cleanup commands when the appropriate flags are set.
func Cleanup(c *dupe.Config, f *cmd.Flags) error {
	if c == nil {
		return dupe.ErrNilConfig
	}
	w := os.Stdout
	switch {
	case f == nil:
		return cmd.ErrNilFlag
	case f.Sensen == nil:
		return fmt.Errorf("%w: sensen", cmd.ErrNilFlag)
	case f.Rm == nil:
		return fmt.Errorf("%w: rm", cmd.ErrNilFlag)
	case f.RmPlus == nil:
		return fmt.Errorf("%w: rmplus", cmd.ErrNilFlag)
	case f.Yes == nil:
		return fmt.Errorf("%w: yes", cmd.ErrNilFlag)
	case *f.Rm:
		return runRemove(w, c)
	case *f.RmPlus:
		return runRemovePlus(w, c)
	case *f.Sensen:
		return runSensen(w, c)
	default:
		return nil
	}
}

// runRemove deletes duplicate files.
// This is intended fpr the -rm flag.
func runRemove(w io.Writer, c *dupe.Config) error {
	s, err := c.DelDupeFiles()
	if err != nil {
		return err
	}
	_, _ = fmt.Fprint(w, s)
	return nil
}

// runRemovePlus deletes duplicate files and empty directories.
// This is intended for the -rm+ flag.
func runRemovePlus(w io.Writer, c *dupe.Config) error {
	s, err := c.DelDupeFiles()
	if err != nil {
		return err
	}
	_, _ = fmt.Fprint(w, s)
	if err := c.DelEmptyDirs(w); err != nil {
		return err
	}
	return nil
}

// runSensen does the following and is intended for the -sensen flag.
//   - deletes duplicate files
//   - delete directories except those with MS-DOS app
//   - delete empty directories
func runSensen(w io.Writer, c *dupe.Config) error {
	s, err := c.DelDupeFiles()
	if err != nil {
		return err
	}
	_, _ = fmt.Fprint(w, s)
	// deleted, err := c.Removes()
	deleted, err := c.DelDirsExcept()
	if err != nil {
		return err
	}
	for _, path := range deleted {
		dupe.PrintRM(path, ErrNoRM)
	}
	// if err := c.Clean(w); err != nil {
	if err := c.DelEmptyDirs(w); err != nil {
		return err
	}
	// v1.01
	//
	// if *f.Sensen {
	// 	removes, err := c.Removes()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	for _, name := range removes {
	// 		dupe.PrintRM(name, ErrNoRM)
	// 	}
	// }
	// if *f.RmPlus || *f.Sensen {
	// 	if err := c.Clean(w); err != nil {
	// 		return err
	// 	}
	// }
	return nil
}

// Check parses the arguments of the dupe command.
func Check(expected int, args ...string) {
	count := len(args)
	w := os.Stdout
	if count < expected {
		printer.StderrCR(ErrNoArgs)
		printl(w, "\nThe dupe command requires a directory or file to check.")
		if runtime.GOOS == winOS {
			printl(w, "The optional bucket can be one or more directories or drive letters.")
		} else {
			printl(w, "The optional bucket can be one or more directory paths.")
		}
		printer.Example("\ndupers dupe <directory or file to check> [buckets to lookup]")
	}
	if count == expected {
		printl(w, color.Warn.Sprint("The database is empty.\n"))
		if runtime.GOOS == winOS {
			printl(w, "This dupe request requires at least one directory or drive letter to lookup.")
		} else {
			printl(w, "This dupe request requires at least one directory to lookup.")
		}
		printl(w, "These lookup directories will be stored to the database as buckets.")
		if len(flag.Args()) > 0 {
			s := fmt.Sprintf("\ndupers dupe %s <one or more directories>\n", flag.Args()[1])
			printer.Example(s)
		}
	}
}

// WalkScanSave both cleans and then updates the buckets with file system changes.
func WalkScanSave(db *bolt.DB, c *dupe.Config, f *cmd.Flags) error {
	if db == nil {
		return bberr.ErrDatabaseNotOpen
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
			ignore(err)
			return nil
		}
	}
	c.Debugger("walk the buckets.")
	return c.WalkDirs(db)
}

func ignore(err error) {
	_, _ = fmt.Fprint(io.Discard, err)
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
			if errors.Is(err, bberr.ErrBucketNotFound) {
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
		return bberr.ErrDatabaseNotOpen
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
		printl(os.Stderr, "The -fast flag cannot be used for this dupe query")
	}
	if fastFlagErr {
		return ErrFast
	}
	return nil
}
