// © Ben Garrett https://github.com/bengarrett/dupers

// Package database interacts with Dupers bbolt database and buckets.
package database

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/bengarrett/dupers/database/internal/bucket"
	"github.com/bengarrett/dupers/internal/out"
	"github.com/dustin/go-humanize"
	"github.com/gookit/color"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

type (
	// Bucket is the absolute path to the directory that is used as the bucket name.
	Bucket string
	// Filepath is the absolute path to a file used as a map key.
	Filepath string
	// Lists are a collection of fetched filepaths and their SHA256 checksums.
	Lists map[Filepath][32]byte
	// Matches are a collection of fetched filepaths and the bucket they were sourced from.
	Matches map[Filepath]Bucket
)

const (
	PrivateFile fs.FileMode = 0o600
	PrivateDir  fs.FileMode = 0o700

	NotFound = "This is okay as one will be created when using the dupe or search commands."
	Timeout  = 3 * time.Second

	backupTime = "20060102-150405"
	boltName   = "dupers.db"
	csvName    = "dupers-export.csv"
	subdir     = "dupers"
	winOS      = "windows"
	tabPadding = 4
	tabWidth   = 8
)

var (
	ErrBucketNotFound = bolt.ErrBucketNotFound
	ErrDBClean        = errors.New("database has nothing to clean")
	ErrDBCompact      = errors.New("database compression has not reduced the size")
	ErrDBEmpty        = errors.New("database is empty and contains no items")
	ErrDBNotFound     = errors.New("database file does not exist")
	ErrDBZeroByte     = errors.New("database is a zero byte file")
)

var (
	TestMode = false //nolint:gochecknoglobals
)

// Abs returns an absolute representation of the named bucket.
func Abs(name string) (string, error) {
	return bucket.Abs(name)
}

// AbsB returns an absolute representation of the named bucket.
func AbsB(name string) ([]byte, error) {
	s, err := bucket.Abs(name)
	return []byte(s), err
}

// All returns every stored bucket within the database.
func All(db *bolt.DB) (names []string, err error) {
	if db == nil {
		db, err = OpenRead()
		if err != nil {
			return nil, err
		}
		defer db.Close()
	}

	if errV := db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			if v := tx.Bucket(name); v == nil {
				return fmt.Errorf("%w: %s", ErrBucketNotFound, string(name))
			}
			names = append(names, string(name))
			return nil
		})
	}); errV != nil {
		return nil, errV
	}

	return names, nil
}

// Check the size and existence of the database file.
func Check() error {
	path, err := DB()
	if err != nil {
		return err
	}
	w := os.Stdout
	i, err1 := os.Stat(path)
	if os.IsNotExist(err1) {
		out.ErrCont(ErrDBNotFound)
		fmt.Fprintf(w, "\n%s\nThe database will be located at: %s\n", NotFound, path)
		return ErrDBNotFound // 0
	} else if err1 != nil {
		return err
	}
	if i.Size() == 0 {
		out.ErrCont(ErrDBZeroByte)
		s := "This error occures when dupers cannot save any data to the file system."
		fmt.Fprintf(w, "\n%s\nThe database is located at: %s\n", s, path)
		return ErrDBZeroByte // 1
	}
	return nil
}

// Exist returns an error if the bucket does not exists in the database.
func Exist(bucket string, db *bolt.DB) error {
	var err error
	if db == nil {
		db, err = OpenRead()
		if err != nil {
			return err
		}
		defer db.Close()
	}

	return db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrBucketNotFound
		}
		return nil
	})
}

// Clean the stale items from database buckets.
// Stale items are file pointers that no longer exist on the host file system.
func Clean(quiet, debug bool, buckets ...string) error {
	cleanDebug(debug, buckets)
	path, err := DB()
	if err != nil {
		return err
	}
	if debug {
		out.PBug("database path: " + path)
	}
	db, err := bolt.Open(path, PrivateFile, write())
	if err != nil {
		return err
	}
	defer db.Close()
	buckets, err = cleaner(buckets, debug, db)
	if err != nil {
		return err
	}
	cnt, errs, finds := 0, 0, 0
	total, err := bucket.Total(buckets, db)
	if err != nil {
		return err
	}
	for _, name := range buckets {
		var abs string
		var cont bool
		parser := bucket.Parser{
			Name:  name,
			DB:    db,
			Cnt:   cnt,
			Errs:  errs,
			Debug: debug,
		}
		if cnt, errs, abs, cont = parser.Parse(); cont {
			continue
		}
		cleaner := bucket.Cleaner{
			DB: db, Abs: abs, Debug: debug, Quiet: quiet, Cnt: cnt, Total: total, Finds: finds, Errs: errs,
		}
		cnt, finds, errs = cleaner.Clean()
	}
	if quiet {
		return nil
	}
	if len(buckets) == errs {
		return nil
	}
	w := os.Stdout
	if debug && finds == 0 {
		fmt.Fprintln(w, "")
		return ErrDBClean
	}
	if finds > 0 {
		fmt.Fprintf(w, "\rThe database removed %d stale items\n", finds)
		return nil
	}
	fmt.Fprintln(w, "")
	return nil
}

func cleanDebug(debug bool, buckets []string) {
	if debug {
		out.PBug("running database clean")

		s := fmt.Sprintf("list of buckets:\n%s",
			strings.Join(buckets, "\n"))
		out.PBug(s)
	}
}

func cleaner(buckets []string, debug bool, db *bolt.DB) ([]string, error) {
	if len(buckets) > 0 {
		return buckets, nil
	}

	if debug {
		out.PBug("fetching all buckets")
	}

	var err1 error
	buckets, err1 = All(db)

	if err1 != nil {
		return nil, err1
	}

	if len(buckets) == 0 {
		return nil, ErrDBEmpty
	}

	return buckets, nil
}

// Compact the database by reclaiming internal space.
func Compact(debug bool) error {
	if debug {
		out.PBug("running database compact")
	}
	// active database
	src, err := DB()
	if err != nil {
		return err
	}
	// make a temporary database
	tmp := filepath.Join(os.TempDir(), backup())
	// open both databases
	srcDB, err := bolt.Open(src, PrivateFile, write())
	if err != nil {
		return err
	} else if debug {
		out.PBug("opened original database: " + src)
	}
	defer srcDB.Close()
	tmpDB, err := bolt.Open(tmp, PrivateFile, write())
	if err != nil {
		return err
	} else if debug {
		out.PBug("opened replacement database: " + tmp)
	}
	defer tmpDB.Close()
	// compress and copy the results to the temporary database
	if debug {
		out.PBug("compress and copy databases")
	}
	if errComp := bolt.Compact(tmpDB, srcDB, 0); errComp != nil {
		return errComp
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
		out.PBug(s1)
		s2 := fmt.Sprintf("new database:      %d bytes, %s", tm.Size(), tm.Name())
		out.PBug(s2)
	}
	if err = srcDB.Close(); err != nil {
		out.ErrFatal(err)
	}
	if cp, err := CopyFile(tmp, src); err != nil {
		return err
	} else if debug {
		s := fmt.Sprintf("copied %d bytes to: %s", cp, src)
		out.PBug(s)
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
	db, err := OpenRead()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if buckets, err = checker(buckets); err != nil {
		return nil, err
	}
	finds := make(Matches)
	for _, bucket := range buckets {
		abs, err := AbsB(bucket)
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
				k := compareKey(key, noCase)
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

func checker(buckets []string) ([]string, error) {
	if len(buckets) != 0 {
		return buckets, nil
	}
	all, err := All(nil)
	if err != nil {
		return nil, err
	}
	if len(all) == 0 {
		return nil, ErrDBEmpty
	}
	return all, nil
}

func compareKey(key []byte, noCase bool) []byte {
	k := key
	if noCase {
		k = bytes.ToLower(key)
	}
	return k
}

// Count the number of records in the bucket.
func Count(name string, db *bolt.DB) (items int, err error) {
	if db == nil {
		db, err = OpenRead()
		if err != nil {
			return 0, err
		}
		defer db.Close()
	}
	return bucket.Count(name, db)
}

// DB returns the absolute path of the database.
func DB() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir, err = os.UserHomeDir()
		if err != nil {
			return "", err
		}
	}
	dir = filepath.Join(dir, subdir)
	if TestMode {
		dir = filepath.Join(dir, "test")
	}
	// create database directory if it doesn't exist
	if _, err = os.Stat(dir); os.IsNotExist(err) {
		if errMk := os.MkdirAll(dir, PrivateDir); errMk != nil {
			return "", fmt.Errorf("cannot create database directory: %w: %s", errMk, dir)
		}
	}
	// create a new database if it doesn't exist, this prevents
	// posix system returning the error a "bad file descriptor" when reading
	path := filepath.Join(dir, boltName)
	i, errP := os.Stat(path)
	if os.IsNotExist(errP) {
		if errDB := Create(path); errDB != nil {
			return "", errDB
		}
		return path, nil
	} else if errP != nil {
		return "", errP
	}
	// recreate an empty database if it is a zero byte file
	if i.Size() == 0 {
		if errRM := os.Remove(path); errRM != nil {
			return "", errRM
		}
		if errDB := Create(path); errDB != nil {
			return "", errDB
		}
	}
	return path, nil
}

// Create a new database at the given path.
func Create(path string) error {
	db, err := bolt.Open(path, PrivateFile, write())
	if err != nil {
		return fmt.Errorf("could not create a new database: %w: %s", err, path)
	}
	defer db.Close()
	return nil
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
	if runtime.GOOS != winOS {
		fmt.Fprintf(w, " (%v)", s.Mode())
	}
	fmt.Fprintf(w, "\n")
	var bucketsB int
	w, bucketsB, err = info(w, path)
	if err != nil {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "\tDatabase error:\t%s\n", err.Error())
	}
	const oneAndAHalf, oneMB = 1.5, 1_000_000
	tooBig := int64(float64(bucketsB) * oneAndAHalf)
	if s.Size() > oneMB && s.Size() > tooBig {
		fmt.Fprintln(w, color.Notice.Sprint("\nTo reduce the size of the database:"))
		fmt.Fprintln(w, color.Debug.Sprint("dupers backup && dupers clean"))
	}
	w.Flush()
	return b.String(), nil
}

func info(w *tabwriter.Writer, name string) (*tabwriter.Writer, int, error) {
	type (
		vals struct {
			items int
			size  string
		}
		item map[string]vals
	)
	db, err := bolt.Open(name, PrivateFile, read())
	if err != nil {
		return w, 0, err
	}
	defer db.Close()
	ro := color.Green.Sprint("OK")
	if !db.IsReadOnly() {
		ro = color.Danger.Sprint("NO")
	}
	fmt.Fprintf(w, "\tRead only mode:\t%s\n", ro)
	items, cnt, sizes := make(item), 0, 0
	if err := db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			v := tx.Bucket(name)
			if v == nil {
				return fmt.Errorf("%w: %s", ErrBucketNotFound, string(name))
			}
			cnt++
			sizes += v.Stats().LeafAlloc
			items[string(name)] = vals{v.Stats().KeyN, humanize.Bytes(uint64(v.Stats().LeafAlloc))}
			return nil
		})
	}); err != nil {
		return nil, 0, err
	}
	fmt.Fprintf(w, "Buckets:        %s\n\n", color.Primary.Sprint(cnt))
	if cnt == 0 {
		// exit when no buckets exist
		return w, sizes, nil
	}
	tab := tabwriter.NewWriter(w, 0, 0, tabPadding, ' ', tabwriter.AlignRight)
	fmt.Fprintf(tab, "Items\tSize\t\tBucket %s\n", color.Secondary.Sprint("(absolute path)"))
	p := message.NewPrinter(language.English)
	for i, b := range items {
		p.Fprintf(tab, "%d\t%s\t\t%v\n", number.Decimal(b.items), b.size, i)
	}
	if err := tab.Flush(); err != nil {
		return w, 0, err
	}
	return w, sizes, nil
}

// IsEmpty returns true when the database has no buckets.
func IsEmpty() (bool, error) {
	path, err := DB()
	if err != nil {
		return true, err
	}
	db, err := bolt.Open(path, PrivateFile, write())
	if err != nil {
		return true, err
	}
	defer db.Close()
	cnt := 0
	if errV := db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			cnt++
			return nil
		})
	}); errV != nil {
		return true, errV
	}
	if cnt == 0 {
		return true, nil
	}
	return false, nil
}

// List returns the filepaths and SHA256 checksums stored in the bucket.
func List(bucket string, db *bolt.DB) (ls Lists, err error) {
	if db == nil {
		db, err = OpenRead()
		if err != nil {
			return nil, err
		}
		defer db.Close()
	}
	lists := make(Lists)
	if errV := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrBucketNotFound
		}
		h := [32]byte{}
		err = b.ForEach(func(k, v []byte) error {
			copy(h[:], v)
			lists[Filepath(k)] = h
			return nil
		})
		return err
	}); errV != nil {
		return nil, errV
	}
	return lists, nil
}

// Rename the named bucket in the database to use a new directory path.
func Rename(name, newName string) error {
	db, err := OpenWrite()
	if err != nil {
		return err
	}
	defer db.Close()
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(name))
		if b == nil {
			return ErrBucketNotFound
		}
		ren, errRen := tx.CreateBucket([]byte(newName))
		if errRen != nil {
			return errRen
		}
		if errPut := b.ForEach(func(k, v []byte) error {
			return ren.Put(k, v)
		}); errPut != nil {
			return errPut
		}
		return tx.DeleteBucket([]byte(name))
	})
}

// RM removes the named bucket from the database.
func RM(name string) error {
	db, err := OpenWrite()
	if err != nil {
		return err
	}
	defer db.Close()
	return db.Update(func(tx *bolt.Tx) error {
		if b := tx.Bucket([]byte(name)); b == nil {
			return ErrBucketNotFound
		}
		return tx.DeleteBucket([]byte(name))
	})
}
