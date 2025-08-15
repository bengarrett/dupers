// © Ben Garrett https://github.com/bengarrett/dupers
package bucket

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/bengarrett/dupers/internal/printer"
	"github.com/gookit/color"
	bolt "go.etcd.io/bbolt"
	bberr "go.etcd.io/bbolt/errors"
)

var (
	ErrBucketAsFile = errors.New("bucket points to a file, not a directory")
	ErrBucketExists = errors.New("bucket already exists in the database")
	ErrBucketNotDir = errors.New("bucket path is not a directory")
	ErrBucketPath   = errors.New("directory used by the bucket does not exist on your system")
	ErrBucketSkip   = errors.New("bucket directory does not exist")
	ErrNilBucket    = errors.New("bucket cannot be an empty directory")
)

const query = "What bucket name do you wish to use"

type Cleaner struct {
	Name  string // Name of the bucket.
	Debug bool   // Debug spams technobabble to stdout.
	Quiet bool   // Quiet the feedback sent to stdout.
	Total int    // Total items handled.
	Items int    // Items is the sum of the bucket items.
	Finds int    // Finds is the sum of the cleaned items.
	Errs  int    // Errs is the sum of the items that could not be cleaned.
}

// Clean the stale items from database buckets.
//
// Returned are the counted items, the finds and the number of errors.
func (c *Cleaner) Clean(db *bolt.DB) (int, int, int, error) {
	if db == nil {
		return 0, 0, 0, bberr.ErrDatabaseNotOpen
	}
	if err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(c.Name))
		if b == nil {
			return fmt.Errorf("%w: %s", bberr.ErrBucketNotFound, c.Name)
		}
		err := b.ForEach(func(k, v []byte) error {
			c.Items++
			printStat(c.Debug, c.Quiet, c.Items, c.Total, k)
			if _, errS := os.Stat(string(k)); errS != nil {
				f := string(k)
				if st, err2 := os.Stat(filepath.Dir(f)); err2 == nil {
					if !st.IsDir() && st.Size() > 0 {
						return nil
					}
				}
				printer.Debug(c.Debug, fmt.Sprintf("%s: %s", k, errS))
				if errUp := db.Update(func(tx *bolt.Tx) error {
					return tx.Bucket([]byte(c.Name)).Delete(k)
				}); errUp != nil {
					return errUp
				}
				c.Finds++
				return nil
			}
			return nil
		})
		if err != nil {
			c.Errs++
			printer.StderrCR(err)
		}
		return nil
	}); err != nil {
		c.Errs++

		printer.StderrCR(err)
	}

	return c.Items, c.Finds, c.Errs, nil
}

type Parser struct {
	Name  string // Name of the bucket to parse.
	Debug bool   // Debug spams technobabble to stdout.
	Items int    // Items is the sum of the items.
	Errs  int    // Errs is the sum of the items that could not be parse.
}

// Parse the parse returns the items and errors count and the absolute path.
// The return boolean is a debug notifier.
func (p *Parser) Parse(db *bolt.DB) (int, int, string, bool) {
	if db == nil {
		printer.StderrCR(bberr.ErrDatabaseNotOpen)
		return -1, -1, "", true
	}
	abs, err := Abs(p.Name)
	if err != nil {
		printer.StderrCR(err)
		return p.Items, p.Errs, "", true
	}
	printer.Debug(p.Debug, "bucket: "+abs)

	// check the bucket directory exists on the file system
	fi, errS := os.Stat(abs)
	switch {
	case os.IsNotExist(errS):
		printer.StderrCR(fmt.Errorf("%w: %s", ErrBucketSkip, abs))
		p.Errs++

		if i, errc := Count(db, p.Name); errc == nil {
			p.Items += i
		}

		return p.Items, p.Errs, "", true
	case errS != nil:
		printer.StderrCR(errS)
		p.Errs++

		return p.Items, p.Errs, "", true
	case !fi.IsDir():
		printer.StderrCR(fmt.Errorf("%w: %s", ErrBucketAsFile, abs))
		p.Errs++
		if i, errc := Count(db, p.Name); errc == nil {
			p.Items += i
		}

		return p.Items, p.Errs, "", true
	}
	return p.Items, p.Errs, abs, false
}

// Abs returns an absolute representation of the named bucket.
func Abs(name string) (string, error) {
	if name == "" {
		return "", ErrNilBucket
	}
	s, err := filepath.Abs(name)
	if err != nil {
		return "", err
	}

	if runtime.GOOS == "windows" {
		return strings.ToLower(s), nil
	}

	return s, nil
}

func printStat(debug, quiet bool, cnt, total int, k []byte) {
	if !debug && !quiet {
		fmt.Fprintf(os.Stdout, "%s", printer.Status(cnt, total, printer.Check))
	}
	printer.Debug(debug, "clean: "+string(k))
}

// Count the number of records in the bucket.
func Count(db *bolt.DB, name string) (int, error) {
	if db == nil {
		return 0, bberr.ErrDatabaseNotOpen
	}
	items := 0
	return items, db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(name))
		if b == nil {
			return bberr.ErrBucketNotFound
		}
		var err error
		items, err = count(db, b)
		if err != nil {
			return err
		}
		return nil
	})
}

func count(db *bolt.DB, b *bolt.Bucket) (int, error) {
	if db == nil {
		return 0, bberr.ErrDatabaseNotOpen
	}
	if b == nil {
		return 0, bberr.ErrBucketNotFound
	}
	records := 0
	return records, db.View(func(tx *bolt.Tx) error {
		if b == nil {
			return bberr.ErrBucketNotFound
		}
		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			records++
		}
		return nil
	})
}

// Rename prompts for confirmation for the use of the named bucket.
func Rename(name string, assumeYes bool) string {
	printer.StderrCR(ErrBucketExists)
	w := os.Stdout
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Import bucket name: %s", color.Debug.Sprint(name))
	fmt.Fprintln(w)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "The existing data in this bucket will overridden and any new data will be appended.")
	if printer.AskYN("Do you want to continue using this bucket", assumeYes, printer.Yes) {
		return name
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Please choose a new bucket, which must be an absolute directory path.")
	return printer.Prompt(query)
}

// Stats checks the validity of the named bucket and prompts for user confirmation on errors.
func Stats(name string, assumeYes bool) bool {
	for {
		fmt.Fprintln(os.Stdout)
		if name = Stat(name, assumeYes, false); name != "" {
			return true
		}
	}
}

func Stat(name string, assumeYes, test bool) string {
	w := os.Stdout
	printName := func() {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Import bucket directory: %s", color.Debug.Sprint(name))
		fmt.Fprintln(w)
		fmt.Fprintln(w)
	}
	if name == "" {
		return ""
	}
	name = strings.TrimSpace(name)
	abs, err := Abs(name)
	if err != nil {
		printer.StderrCR(fmt.Errorf("%w: %s", err, name))
		return ""
	}
	s, err := os.Stat(abs)
	if errors.Is(err, os.ErrNotExist) {
		printer.StderrCR(ErrBucketPath)
		printName()
		fmt.Fprintln(w, "You may still run dupe checks and searches without the actual files on your system.")
		fmt.Fprintln(w, "Choosing no will prompt for a new bucket.")
		if !test {
			if printer.AskYN("Do you want to continue using this bucket", assumeYes, printer.Yes) {
				return abs
			}
			return printer.Prompt(query)
		}
		return ""
	} else if err == nil && !s.IsDir() {
		err = ErrBucketNotDir
	}
	if err != nil {
		printer.StderrCR(ErrBucketNotDir)
		printName()
		fmt.Fprintln(w, "You cannot use this path as a bucket, please choose an absolute directory path.")
		if !test {
			return printer.Prompt(query)
		}
		return ""
	}
	return abs
}

// Total returns the sum total of the items in the named buckets.
func Total(db *bolt.DB, buckets []string) (int, error) {
	if db == nil {
		return 0, bberr.ErrDatabaseNotOpen
	}
	if len(buckets) == 0 {
		return 0, ErrNilBucket
	}

	count := 0

	for _, bucket := range buckets {
		abs, err := Abs(bucket)
		if err != nil {
			return 0, err
		}

		items, err := Count(db, abs)
		if err != nil {
			return 0, err
		}

		count += items
	}

	return count, nil
}
