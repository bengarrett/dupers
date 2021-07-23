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
	"strings"
	"text/tabwriter"
	"time"

	"github.com/bengarrett/dupers/lib/out"
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
	FileMode   fs.FileMode = 0600
	backupTime             = "20060102-150405"
	dbName                 = "dupers.db"
	dbPath                 = "dupers"
)

var (
	ErrNoBucket  = errors.New("bucket does not exist")
	ErrDB        = errors.New("database does not exist")
	ErrDBClean   = errors.New("database had nothing to clean")
	ErrDBCompact = errors.New("database compression has not reduced the size")

	testMode = false // nolint: gochecknoglobals
)

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

	written, err = copyFile(src, name)
	if err != nil {
		return "", 0, err
	}
	return name, written, nil
}

func backupName() string {
	now, ext := time.Now().Format(backupTime), filepath.Ext(dbName)
	return fmt.Sprintf("%s-backup-%s%s", strings.TrimSuffix(dbName, ext), now, ext)
}

func copyFile(src, dest string) (int64, error) {
	// read source
	f, err := os.Open(src)
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
	db, err := bolt.Open(path, FileMode, &bolt.Options{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if err1 := db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			v := tx.Bucket(name)
			if v == nil {
				return fmt.Errorf("%w: %s", ErrNoBucket, string(name))
			}
			names = append(names, string(name))
			return nil
		})
	}); err1 != nil {
		return nil, err1
	}
	return names, nil
}

// Clean the stale items from all database buckets.
// Stale items are file pointers that no longer exist on the host file system.
func Clean(quiet, debug bool) error { // nolint: gocyclo
	if debug {
		out.Bug("running database clean")
	}
	buckets, err := AllBuckets()
	if err != nil {
		return err
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
	db, err := bolt.Open(path, FileMode, nil)
	if err != nil {
		return err
	}
	defer db.Close()
	cnt := 0
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
				return ErrNoBucket
			}
			err = b.ForEach(func(k, v []byte) error {
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
					cnt++
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
	if cnt == 0 {
		return ErrDBClean
	}
	fmt.Printf("The database removed %d stale items", cnt)
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
	srcDB, err := bolt.Open(src, FileMode, nil)
	if err != nil {
		return err
	} else if debug {
		out.Bug("opened original database: " + src)
	}
	defer srcDB.Close()
	tmpDB, err := bolt.Open(tmp, FileMode, nil)
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
	if cp, err := copyFile(tmp, src); err != nil {
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
	db, err := bolt.Open(path, FileMode, &bolt.Options{ReadOnly: true})
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
				return ErrNoBucket
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
		}); errors.Is(err, ErrNoBucket) {
			return nil, fmt.Errorf("%w: '%s'", err, abs)
		} else if err != nil {
			return nil, err
		}
	}
	return &finds, nil
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
		if err1 := os.MkdirAll(dir, 0700); err != nil {
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
	w.Init(&b, 0, 8, 0, '\t', 0)
	fmt.Fprintf(w, "\tLocation:\t%s\n", path)
	s, err := os.Stat(path)
	if os.IsNotExist(err) {
		fmt.Fprintln(w, "\nThis is okay, one will be created during the next dupe or bucket scan.")
		w.Flush()
		return b.String(), ErrDB
	} else if err != nil {
		w.Flush()
		return b.String(), err
	}
	fmt.Fprintf(w, "\tFile size:\t%s\n",
		color.Primary.Sprint(humanize.Bytes(uint64(s.Size()))))
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
	db, err := bolt.Open(name, FileMode, &bolt.Options{ReadOnly: true})
	if err != nil {
		return w, err
	}
	defer db.Close()
	ro := color.Green.Sprint("OK")
	if !db.IsReadOnly() {
		ro = color.Danger.Sprint("NO")
	}
	fmt.Fprintf(w, "\tRead only mode:\t%s\n", ro)
	fmt.Fprintln(w, "Buckets:")
	err = db.View(func(tx *bolt.Tx) error {
		cnt := 0
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			v := tx.Bucket(name)
			if v == nil {
				return fmt.Errorf("%w: %s", ErrNoBucket, string(name))
			}
			cnt++
			fmt.Fprintln(w)
			fmt.Fprintf(w, "\t%s\n", string(name))
			items := v.Stats().KeyN
			if items == 0 {
				fmt.Fprintf(w, "\t\t   ⤷ is empty")
				return nil
			}
			fmt.Fprintf(w, "\t\t %s %s %s %s\n", color.Secondary.Sprint("⤷"),
				color.Primary.Sprint(items), color.Secondary.Sprint("items,"), color.Primary.Sprint(humanize.Bytes(uint64(v.Stats().LeafAlloc))))
			return nil
		})
	})
	return w, err
}

// IsEmpty returns true if the database has no buckets.
func IsEmpty() (bool, error) {
	path, err := DB()
	if err != nil {
		return true, err
	}
	db, err := bolt.Open(path, FileMode, nil)
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
	db, err := bolt.Open(path, FileMode, &bolt.Options{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer db.Close()
	ls = make(Lists)
	if err1 := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrNoBucket
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
	db, err := bolt.Open(path, FileMode, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(name))
		if b == nil {
			return ErrNoBucket
		}
		return tx.DeleteBucket([]byte(name))
	})
}

// Seek searches a bucket for an exact SHA256 checksum match.
func Seek(sum [32]byte, bucket string) (finds []string, records int, err error) {
	path, err := DB()
	if err != nil {
		return nil, 0, err
	}
	db, err := bolt.Open(path, FileMode, &bolt.Options{ReadOnly: true})
	if err != nil {
		return nil, 0, err
	}
	defer db.Close()
	return finds, records, db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrNoBucket
		}
		h := sum[:]
		err = b.ForEach(func(k, v []byte) error {
			records++
			if bytes.Equal(v, h) {
				finds = append(finds, string(k))
			}
			return nil
		})
		return err
	})
}
