// © Ben Garrett https://github.com/bengarrett/dupers

// Package database interacts with Dupers bbolt database and buckets.
package database

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/bengarrett/dupers/out"
	"github.com/dustin/go-humanize"
	"github.com/gookit/color"
	bolt "go.etcd.io/bbolt"
)

type (
	// Bucket is the absolute path to a directory that's used as the bucket name.
	Bucket string
	// Filepath is the absolute path to a file used as a map key.
	Filepath string
	// Lists are a collection of fetched filepaths and their SHA256 checksums.
	Lists map[Filepath][32]byte
	// Matches are a collection of fetched filepaths and the bucket they were sourced from.
	Matches map[Filepath]Bucket
)

const (
	PrivateFile fs.FileMode = 0600
	PrivateDir  fs.FileMode = 0700

	NotFound   = "This is okay as one will be created when using the dupe or search commands."
	backupTime = "20060102-150405"
	dbName     = "dupers.db"
	dbPath     = "dupers"
	tabPadding = 4
	tabWidth   = 8
)

var (
	ErrBucketNotFound = bolt.ErrBucketNotFound
	ErrDBClean        = errors.New("database has nothing to clean")
	ErrDBCompact      = errors.New("database compression has not reduced the size")
	ErrDBNotFound     = errors.New("database file does not exist")
	ErrDBZeroByte     = errors.New("database is a zero byte file")

	testMode = false // nolint: gochecknoglobals
)

// Abs returns an absolute representation of the named bucket.
func Abs(bucket string) ([]byte, error) {
	s, err := filepath.Abs(bucket)
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
}

// AllBuckets lists all the stored bucket names in the database.
func AllBuckets() (names []string, err error) {
	path, err := DB()
	if err != nil {
		return nil, err
	}
	db, err := bolt.Open(path, PrivateFile, &bolt.Options{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if err1 := db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			v := tx.Bucket(name)
			if v == nil {
				return fmt.Errorf("%w: %s", ErrBucketNotFound, string(name))
			}
			names = append(names, string(name))
			return nil
		})
	}); err1 != nil {
		return nil, err1
	}
	return names, nil
}

// Backup makes a copy of the database to the named location.
func Backup() (name string, written int64, err error) {
	src, err := DB()
	if err != nil {
		return "", 0, err
	}

	dir, err := os.UserHomeDir()
	if err != nil {
		return "", 0, err
	}
	name = filepath.Join(dir, backupName())

	written, err = CopyFile(src, name)
	if err != nil {
		return "", 0, err
	}
	return name, written, nil
}

func backupName() string {
	now, ext := time.Now().Format(backupTime), filepath.Ext(dbName)
	return fmt.Sprintf("%s-backup-%s%s", strings.TrimSuffix(dbName, ext), now, ext)
}

// Clean the stale items from database buckets.
// Stale items are file pointers that no longer exist on the host file system.
func Clean(quiet, debug bool, buckets ...string) error { // nolint: gocyclo
	if debug {
		out.Bug("running database clean")
	}
	var err error
	if len(buckets) == 0 {
		buckets, err = AllBuckets()
		if err != nil {
			return err
		}
	}
	if debug {
		s := fmt.Sprintf("list of buckets:\n%s", strings.Join(buckets, "\n"))
		out.Bug(s)
	}
	path, err := DB()
	if err != nil {
		return err
	}
	if debug {
		out.Bug("database path: " + path)
	}
	db, err := bolt.Open(path, PrivateFile, nil)
	if err != nil {
		return err
	}
	defer db.Close()
	cnt, finds, total := 0, 0, 0
	for _, bucket := range buckets {
		abs, err := Abs(bucket)
		if err != nil {
			out.ErrCont(err)
		} else if debug {
			out.Bug("bucket: " + string(abs))
		}
		if err = db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket(abs)
			if b == nil {
				return ErrBucketNotFound
			}
			total, err = Count(b, db)
			if err != nil {
				return err
			}
			err = b.ForEach(func(k, v []byte) error {
				cnt++
				if !debug && !quiet {
					fmt.Printf("\rChecking %d of %d items", cnt, total)
				}
				if debug {
					out.Bug("clean: " + string(k))
				}
				if _, err1 := os.Stat(string(k)); err1 != nil {
					f := string(k)
					if st, err2 := os.Stat(filepath.Dir(f)); err2 == nil {
						if !st.IsDir() && st.Size() > 0 {
							return nil
						}
					}
					if debug {
						s := fmt.Sprintf("%s: %s", k, err1)
						out.Bug(s)
					}
					if err2 := db.Update(func(tx *bolt.Tx) error {
						return tx.Bucket(abs).Delete(k)
					}); err2 != nil {
						return err2
					}
					finds++
					return nil
				}
				return nil
			})
			if err != nil {
				out.ErrCont(err)
			}
			return nil
		}); err != nil {
			out.ErrCont(err)
		}
	}
	if quiet {
		return nil
	}
	if finds == 0 {
		fmt.Println()
		return ErrDBClean
	}
	fmt.Printf("The database removed %d stale items\n", finds)
	return nil
}

// Compact the database by reclaiming space.
func Compact(debug bool) error {
	if debug {
		out.Bug("running database compact")
	}
	// active database
	src, err := DB()
	if err != nil {
		return err
	}
	// make a temporary database
	tmp := filepath.Join(os.TempDir(), backupName())
	// open both databases
	srcDB, err := bolt.Open(src, PrivateFile, nil)
	if err != nil {
		return err
	} else if debug {
		out.Bug("opened original database: " + src)
	}
	defer srcDB.Close()
	tmpDB, err := bolt.Open(tmp, PrivateFile, nil)
	if err != nil {
		return err
	} else if debug {
		out.Bug("opened replacement database: " + tmp)
	}
	defer tmpDB.Close()
	// compress and copy the results to the temporary database
	if debug {
		out.Bug("compress and copy databases")
	}
	if err1 := bolt.Compact(tmpDB, srcDB, 0); err1 != nil {
		return err1
	}
	if debug {
		sr, errS := os.Stat(src)
		if errS != nil {
			return errS
		}
		tm, errT := os.Stat(tmp)
		if errT != nil {
			return errT
		}
		s1 := fmt.Sprintf("original database: %d bytes, %s", sr.Size(), sr.Name())
		out.Bug(s1)
		s2 := fmt.Sprintf("new database:      %d bytes, %s", tm.Size(), tm.Name())
		out.Bug(s2)
	}
	if err = srcDB.Close(); err != nil {
		out.ErrFatal(err)
	}
	if cp, err := CopyFile(tmp, src); err != nil {
		return err
	} else if debug {
		s := fmt.Sprintf("copied %d bytes to: %s", cp, src)
		out.Bug(s)
	}
	return nil
}

// Compare finds exact matches of the string contained within the stored filenames and paths.
func Compare(s string, buckets ...string) (*Matches, error) {
	return compare([]byte(s), false, false, buckets...)
}

// CompareBase finds exact matches of the string contained within the stored filenames.
func CompareBase(s string, buckets ...string) (*Matches, error) {
	return compare([]byte(s), false, true, buckets...)
}

// CompareBaseNoCase finds case insensitive matches of the string contained within the stored filenames.
func CompareBaseNoCase(s string, buckets ...string) (*Matches, error) {
	m, err := compare([]byte(s), true, true, buckets...)
	return m, err
}

// CompareNoCase finds case insensitive matches of the string contained within the stored filenames and paths.
func CompareNoCase(s string, buckets ...string) (*Matches, error) {
	return compare([]byte(s), true, false, buckets...)
}

func compare(term []byte, noCase, base bool, buckets ...string) (*Matches, error) {
	path, err := DB()
	if err != nil {
		return nil, err
	}
	db, err := bolt.Open(path, PrivateFile, &bolt.Options{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if len(buckets) == 0 {
		buckets, err = AllBuckets()
		if err != nil {
			return nil, err
		}
	}
	finds := make(Matches)
	for _, bucket := range buckets {
		abs, err := Abs(bucket)
		if err != nil {
			out.ErrCont(err)
		}

		if err = db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket(abs)
			if b == nil {
				return ErrBucketNotFound
			}
			s := term
			if noCase {
				s = bytes.ToLower(term)
			}
			err = b.ForEach(func(key, _ []byte) error {
				k := key
				if noCase {
					k = bytes.ToLower(key)
				}
				if base {
					k = []byte(filepath.Base(string(k)))
					if bytes.Contains(k, s) {
						finds[Filepath(key)] = Bucket(bucket)
					}
					return nil
				}
				if bytes.Contains(k, s) {
					finds[Filepath(key)] = Bucket(bucket)
				}
				return nil
			})
			return err
		}); errors.Is(err, ErrBucketNotFound) {
			return nil, fmt.Errorf("%w: '%s'", err, abs)
		} else if err != nil {
			return nil, err
		}
	}
	return &finds, nil
}

// CopyFile duplicates the named file to the destination filepath.
func CopyFile(name, dest string) (int64, error) {
	// read source
	f, err := os.Open(name)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	// create backup file
	bu, err := os.Create(dest)
	if err != nil {
		return 0, err
	}
	defer bu.Close()
	// duplicate data
	return io.Copy(bu, f)
}

func Count(b *bolt.Bucket, db *bolt.DB) (int, error) {
	records := 0
	if err := db.View(func(tx *bolt.Tx) error {
		if b == nil {
			return ErrBucketNotFound
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

// DB returns the absolute path of the Bolt database.
func DB() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir, err = os.UserHomeDir()
		if err != nil {
			return "", err
		}
	}
	dir = filepath.Join(dir, dbPath)
	if testMode {
		dir = filepath.Join(dir, "test")
	}
	_, err = os.Stat(dir)
	if os.IsNotExist(err) {
		if err1 := os.MkdirAll(dir, PrivateDir); err != nil {
			return "", err1
		}
	}
	return filepath.Join(dir, dbName), nil
}

// Info returns a printout of the buckets and their statistics.
func Info() (string, error) {
	path, err := DB()
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	w := new(tabwriter.Writer)
	w.Init(&b, 0, tabWidth, 0, '\t', 0)
	fmt.Fprintf(w, "\tLocation:\t%s\n", path)
	s, err := os.Stat(path)
	if os.IsNotExist(err) {
		fmt.Fprintf(w, "\n%s\n", NotFound)
		w.Flush()
		return b.String(), ErrDBNotFound
	} else if err != nil {
		w.Flush()
		return b.String(), err
	}
	fmt.Fprintf(w, "\tModified:\t%s\n", s.ModTime().Local().Format("Jan 2 15:04:05"))
	fmt.Fprintf(w, "\tFile:\t%s",
		color.Primary.Sprint(humanize.Bytes(uint64(s.Size()))))
	if runtime.GOOS != "windows" {
		fmt.Fprintf(w, " (%v)", s.Mode())
	}
	fmt.Fprintf(w, "\n")
	w, err = info(path, w)
	if err != nil {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "\tDatabase error:\t%s\n", err.Error())
	}
	const hundredMB = 100_000_000
	if s.Size() > hundredMB {
		fmt.Fprintln(w, color.Notice.Sprint("\nTo reduce the size of the database:"))
		fmt.Fprintln(w, color.Debug.Sprint("duper backup && duper clean"))
	}
	w.Flush()
	return b.String(), nil
}

func info(name string, w *tabwriter.Writer) (*tabwriter.Writer, error) {
	type (
		vals struct {
			items int
			size  string
		}
		item map[string]vals
	)
	db, err := bolt.Open(name, PrivateFile, &bolt.Options{ReadOnly: true})
	if err != nil {
		return w, err
	}
	defer db.Close()
	ro := color.Green.Sprint("OK")
	if !db.IsReadOnly() {
		ro = color.Danger.Sprint("NO")
	}
	fmt.Fprintf(w, "\tRead only mode:\t%s\n", ro)
	items, cnt := make(item), 0
	if err := db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			v := tx.Bucket(name)
			if v == nil {
				return fmt.Errorf("%w: %s", ErrBucketNotFound, string(name))
			}
			cnt++
			items[string(name)] = vals{v.Stats().KeyN, humanize.Bytes(uint64(v.Stats().LeafAlloc))}
			return nil
		})
	}); err != nil {
		return nil, err
	}
	fmt.Fprintf(w, "Buckets:        %s\n\n", color.Primary.Sprint(cnt))
	tab := tabwriter.NewWriter(w, 0, 0, tabPadding, ' ', tabwriter.AlignRight)
	fmt.Fprintf(tab, "Items\tSize\t\tBucket %s\n", color.Secondary.Sprint("(absolute path)"))
	for i, b := range items {
		fmt.Fprintf(tab, "%d\t%s\t\t%v\n", b.items, b.size, i)
	}
	if err := tab.Flush(); err != nil {
		return w, err
	}
	return w, nil
}

// IsEmpty returns true if the database has no buckets.
func IsEmpty() (bool, error) {
	path, err := DB()
	if err != nil {
		return true, err
	}
	db, err := bolt.Open(path, PrivateFile, nil)
	if err != nil {
		return true, err
	}
	defer db.Close()
	cnt := 0
	if err1 := db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			cnt++
			return nil
		})
	}); err1 != nil {
		return true, err1
	}
	if cnt == 0 {
		return true, nil
	}
	return false, nil
}

// List returns the file paths and SHA256 checksums stored in the bucket.
func List(bucket string) (ls Lists, err error) {
	path, err := DB()
	if err != nil {
		return nil, err
	}
	db, err := bolt.Open(path, PrivateFile, &bolt.Options{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer db.Close()
	ls = make(Lists)
	if err1 := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrBucketNotFound
		}
		h := [32]byte{}
		err = b.ForEach(func(k, v []byte) error {
			copy(h[:], v)
			ls[Filepath(k)] = h
			return nil
		})
		return err
	}); err1 != nil {
		return nil, err1
	}
	return ls, nil
}

// RM removes the named bucket from the database.
func RM(name string) error {
	path, err := DB()
	if err != nil {
		return err
	}
	db, err := bolt.Open(path, PrivateFile, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(name))
		if b == nil {
			return ErrBucketNotFound
		}
		return tx.DeleteBucket([]byte(name))
	})
}

// Seek searches a bucket for an exact SHA256 checksum match.
func Seek(sum [32]byte, bucket string, db *bolt.DB) (finds []string, records int, err error) {
	return finds, records, db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrBucketNotFound
		}
		// Cursor search is sorted and runs slightly (20ms) faster on my machine.
		c, h := b.Cursor(), sum[:]
		for k, v := c.First(); k != nil; k, v = c.Next() {
			records++
			if bytes.Equal(v, h) {
				finds = append(finds, string(k))
			}
		}
		return nil
	})
}
