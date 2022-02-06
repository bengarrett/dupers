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
)

const (
	query = "What bucket name do you wish to use"
)

type Cleaner struct {
	DB    *bolt.DB
	Abs   string
	Debug bool
	Quiet bool
	Cnt   int
	Total int
	Finds int
	Errs  int
}

func (c *Cleaner) Clean() (count int, finds int, errors int) {
	if c.DB == nil {
		return 0, 0, 0
	}
	if err := c.DB.View(func(tx *bolt.Tx) error {
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
				if c.Debug {
					out.PBug(fmt.Sprintf("%s: %s", k, errS))
				}
				if errUp := c.DB.Update(func(tx *bolt.Tx) error {
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

	return c.Cnt, c.Finds, c.Errs
}

type Parser struct {
	Name  string
	DB    *bolt.DB
	Cnt   int
	Errs  int
	Debug bool
}

func (p *Parser) Parse() (int, int, string, bool) {
	abs, err := Abs(p.Name)
	if err != nil {
		out.ErrCont(err)
		return p.Cnt, p.Errs, "", true
	} else if p.Debug {
		out.PBug("bucket: " + abs)
	}
	// check the bucket directory exists on the file system
	fi, errS := os.Stat(abs)
	switch {
	case os.IsNotExist(errS):
		out.ErrCont(fmt.Errorf("%w: %s", ErrBucketSkip, abs))
		p.Errs++

		if i, errc := Count(p.Name, p.DB); errc == nil {
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
		if i, errc := Count(p.Name, p.DB); errc == nil {
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
		fmt.Printf("%s", out.Status(cnt, total, out.Check))
	}
	if debug {
		out.PBug("clean: " + string(k))
	}
}

// Count the number of records in the bucket.
func Count(name string, db *bolt.DB) (items int, err error) {
	if db == nil {
		return 0, bolt.ErrBucketNotFound
	}
	if errV := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(name))
		if b == nil {
			return bolt.ErrBucketNotFound
		}
		items, err = count(b, db)
		return nil
	}); errV != nil {
		return 0, errV
	}
	return items, nil
}

func count(b *bolt.Bucket, db *bolt.DB) (int, error) {
	records := 0
	if err := db.View(func(tx *bolt.Tx) error {
		if b == nil {
			return bolt.ErrBucketNotFound
		}
		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			records++
		}
		return nil
	}); err != nil {
		return records, err
	}
	return records, nil
}

// Rename prompts for confirmation for the use of the named bucket.
func Rename(name string) string {
	out.ErrCont(ErrBucketExists)
	fmt.Printf("\nImport bucket name: %s\n\n", color.Debug.Sprint(name))
	fmt.Println("The existing data in this bucket will overridden and any new data will be appended.")
	if out.YN("Do you want to continue using this bucket", out.Yes) {
		return name
	}
	fmt.Println("\nPlease choose a new bucket, which must be an absolute directory path.")
	return out.Prompt(query)
}

// Stats checks the validity of the named bucket and prompts for user confirmation on errors.
func Stats(name string) bool {
	for {
		fmt.Println()
		if name = Stat(name, false); name != "" {
			return true
		}
	}
}

func Stat(name string, test bool) string {
	printName := func() {
		fmt.Printf("\nImport bucket directory: %s\n\n", color.Debug.Sprint(name))
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
		fmt.Println("You may still run dupe checks and searches without the actual files on your system.")
		fmt.Println("Choosing no will prompt for a new bucket.")
		if !test {
			if out.YN("Do you want to continue using this bucket", out.Yes) {
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
		fmt.Println("You cannot use this path as a bucket, please choose an absolute directory path.")
		if !test {
			return out.Prompt(query)
		}
		return ""
	}
	return abs
}

func Total(buckets []string, db *bolt.DB) (int, error) {
	if db == nil {
		return 0, bolt.ErrBucketNotFound
	}

	count := 0

	for _, bucket := range buckets {
		abs, err := Abs(bucket)
		if err != nil {
			out.ErrCont(err)
			continue
		}

		items, err := Count(abs, db)
		if err != nil {
			return -1, err
		}

		count += items
	}

	return count, nil
}
