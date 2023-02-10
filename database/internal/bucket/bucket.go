// © Ben Garrett https://github.com/bengarrett/dupers
package bucket

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/bengarrett/dupers/internal/out"
	"github.com/gookit/color"
	bolt "go.etcd.io/bbolt"
)

var (
	ErrBucketAsFile = errors.New("bucket points to a file, not a directory")
	ErrBucketExists = errors.New("bucket already exists in the database")
	ErrBucketNotDir = errors.New("bucket path is not a directory")
	ErrBucketPath   = errors.New("directory used by the bucket does not exist on your system")
	ErrBucketSkip   = errors.New("bucket directory does not exist")
	ErrDB           = errors.New("db database cannot be nil")
)

const query = "What bucket name do you wish to use"

type Cleaner struct {
	Abs   string // Absolute path of the bucket.
	Debug bool   // Debug spams technobabble to stdout.
	Quiet bool   // Quiet the feedback sent to stdout.
	Cnt   int    // Cnt is the sum of the items.
	Total int    // Total items handled.
	Finds int    // Finds is the sum of the cleaned items.
	Errs  int    // Errs is the sum of the items that could not be cleaned.
}

// Clean the stale items from database buckets.
func (c *Cleaner) Clean(db *bolt.DB) (items, finds, errors int, err error) {
	if db == nil {
		return 0, 0, 0, bolt.ErrDatabaseNotOpen
	}
	if err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(c.Abs))
		if b == nil {
			return fmt.Errorf("%w: %s", bolt.ErrBucketNotFound, c.Abs)
		}
		err := b.ForEach(func(k, v []byte) error {
			c.Cnt++
			printStat(c.Debug, c.Quiet, c.Cnt, c.Total, k)
			if _, errS := os.Stat(string(k)); errS != nil {
				f := string(k)
				if st, err2 := os.Stat(filepath.Dir(f)); err2 == nil {
					if !st.IsDir() && st.Size() > 0 {
						return nil
					}
				}
				out.DPrint(c.Debug, fmt.Sprintf("%s: %s", k, errS))
				if errUp := db.Update(func(tx *bolt.Tx) error {
					return tx.Bucket([]byte(c.Abs)).Delete(k)
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
			out.ErrCont(err)
		}
		return nil
	}); err != nil {
		c.Errs++

		out.ErrCont(err)
	}

	return c.Cnt, c.Finds, c.Errs, nil
}

type Parser struct {
	Name  string   // Name of the bucket to parse.
	DB    *bolt.DB // Bold database.
	Cnt   int      // Cnt is the sum of the items.
	Errs  int      // Errs is the sum of the items that could not be parse.
	Debug bool     // Debug spams technobabble to stdout.
}

func (p *Parser) Parse() (items int, errs int, name string, debug bool) {
	abs, err := Abs(p.Name)
	if err != nil {
		out.ErrCont(err)
		return p.Cnt, p.Errs, "", true
	}
	out.DPrint(p.Debug, "bucket: "+abs)

	// check the bucket directory exists on the file system
	fi, errS := os.Stat(abs)
	switch {
	case os.IsNotExist(errS):
		out.ErrCont(fmt.Errorf("%w: %s", ErrBucketSkip, abs))
		p.Errs++

		if i, errc := Count(p.DB, p.Name); errc == nil {
			p.Cnt += i
		}

		return p.Cnt, p.Errs, "", true
	case err != nil:
		out.ErrCont(err)
		p.Errs++

		return p.Cnt, p.Errs, "", true
	case !fi.IsDir():
		out.ErrCont(fmt.Errorf("%w: %s", ErrBucketAsFile, abs))
		p.Errs++
		if i, errc := Count(p.DB, p.Name); errc == nil {
			p.Cnt += i
		}

		return p.Cnt, p.Errs, "", true
	}
	return p.Cnt, p.Errs, abs, false
}

// Abs returns an absolute representation of the named bucket.
func Abs(name string) (string, error) {
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
		fmt.Fprintf(os.Stdout, "%s", out.Status(cnt, total, out.Check))
	}
	out.DPrint(debug, "clean: "+string(k))
}

// Count the number of records in the bucket.
func Count(db *bolt.DB, name string) (int, error) {
	if db == nil {
		return 0, bolt.ErrDatabaseNotOpen
	}
	items := 0
	return items, db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(name))
		if b == nil {
			return bolt.ErrBucketNotFound
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
		return 0, bolt.ErrDatabaseNotOpen
	}
	if b == nil {
		return 0, bolt.ErrBucketNotFound
	}
	records := 0
	return records, db.View(func(tx *bolt.Tx) error {
		if b == nil {
			return bolt.ErrBucketNotFound
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
	out.ErrCont(ErrBucketExists)
	w := os.Stdout
	fmt.Fprintf(w, "\nImport bucket name: %s\n\n", color.Debug.Sprint(name))
	fmt.Fprintln(w, "The existing data in this bucket will overridden and any new data will be appended.")
	if out.YN("Do you want to continue using this bucket", assumeYes, out.Yes) {
		return name
	}
	fmt.Fprintln(w, "\nPlease choose a new bucket, which must be an absolute directory path.")
	return out.Prompt(query)
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
		fmt.Fprintf(w, "\nImport bucket directory: %s\n\n", color.Debug.Sprint(name))
	}
	if name == "" {
		return ""
	}
	name = strings.TrimSpace(name)
	abs, err := Abs(name)
	if err != nil {
		out.ErrCont(fmt.Errorf("%w: %s", err, name))
		return ""
	}
	s, err := os.Stat(abs)
	if errors.Is(err, os.ErrNotExist) {
		out.ErrCont(ErrBucketPath)
		printName()
		fmt.Fprintln(w, "You may still run dupe checks and searches without the actual files on your system.")
		fmt.Fprintln(w, "Choosing no will prompt for a new bucket.")
		if !test {
			if out.YN("Do you want to continue using this bucket", assumeYes, out.Yes) {
				return abs
			}
			return out.Prompt(query)
		}
		return ""
	} else if err == nil && !s.IsDir() {
		err = ErrBucketNotDir
	}
	if err != nil {
		out.ErrCont(ErrBucketNotDir)
		printName()
		fmt.Fprintln(w, "You cannot use this path as a bucket, please choose an absolute directory path.")
		if !test {
			return out.Prompt(query)
		}
		return ""
	}
	return abs
}

// Total returns the sum total of the items in the named buckets.
func Total(db *bolt.DB, buckets []string) (int, error) {
	if db == nil {
		return 0, bolt.ErrDatabaseNotOpen
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
