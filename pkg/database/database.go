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

	"github.com/bengarrett/dupers/internal/out"
	"github.com/bengarrett/dupers/pkg/database/internal/bucket"
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
	ErrClean     = errors.New("database has nothing to clean")
	ErrCompact   = errors.New("database compression has not reduced the size")
	ErrEmpty     = errors.New("database is empty and contains no items")
	ErrNotFound  = errors.New("database file does not exist")
	ErrSameNames = errors.New("bucket target is the same as the bucket name")
	ErrZeroByte  = errors.New("database is a zero byte file")

	ErrNoTerm = errors.New("cannot compare an empty term")
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
func All(db *bolt.DB) ([]string, error) {
	if db == nil {
		return nil, bolt.ErrDatabaseNotOpen
	}
	var names []string
	if err := db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			if v := tx.Bucket(name); v == nil {
				return fmt.Errorf("%w: %s", bolt.ErrBucketNotFound, string(name))
			}
			names = append(names, string(name))
			return nil
		})
	}); err != nil {
		return nil, err
	}

	return names, nil
}

// Check returns size and existence of the database file.
func Check() (int64, error) {
	path, err := DB()
	if err != nil {
		return 0, err
	}
	w := os.Stdout
	i, err1 := os.Stat(path)
	if os.IsNotExist(err1) {
		out.StderrCR(ErrNotFound)
		fmt.Fprintf(w, "\n%s\nThe database will be located at: %s\n", NotFound, path)
		return 0, ErrNotFound // 0
	} else if err1 != nil {
		return 0, err
	}
	if i.Size() == 0 {
		out.StderrCR(ErrZeroByte)
		s := "This error occures when dupers cannot save any data to the file system."
		fmt.Fprintf(w, "\n%s\nThe database is located at: %s\n", s, path)
		return 0, ErrZeroByte // 1
	}
	return i.Size(), nil
}

// Exist returns an error if the bucket does not exists in the database.
func Exist(db *bolt.DB, bucket string) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	return db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return bolt.ErrBucketNotFound
		}
		return nil
	})
}

// Clean the stale items from database buckets.
// Stale items are file pointers that no longer exist on the host file system.
func Clean(db *bolt.DB, quiet, debug bool, buckets ...string) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	cleanDebug(debug, buckets)

	out.DPrint(debug, fmt.Sprintf("cleaner of buckets: %s", buckets))
	cleaned, err := cleaner(db, debug, buckets)
	if err != nil {
		return err
	}

	cnt, errs, finds := 0, 0, 0
	total, err := bucket.Total(db, cleaned)
	if err != nil {
		return err
	}

	for _, name := range cleaned {
		var abs string
		var cont bool
		parser := bucket.Parser{
			Name:  name,
			Items: cnt,
			Errs:  errs,
			Debug: debug,
		}
		if cnt, errs, abs, cont = parser.Parse(db); cont {
			continue
		}
		cleaner := bucket.Cleaner{
			Name:  abs,
			Debug: debug,
			Quiet: quiet,
			Items: cnt,
			Total: total,
			Finds: finds,
			Errs:  errs,
		}
		cnt, finds, errs, err = cleaner.Clean(db)
		if err != nil {
			return err
		}
	}
	if quiet {
		return nil
	}
	if len(cleaned) == errs {
		return nil
	}
	w := os.Stdout
	if debug && finds == 0 {
		fmt.Fprintln(w, "")
		return ErrClean
	}
	if finds > 0 {
		fmt.Fprintf(w, "\rThe database removed %d stale items\n", finds)
		return nil
	}
	fmt.Fprintln(w, "")
	return nil
}

func cleanDebug(debug bool, buckets []string) {
	if !debug {
		return
	}
	out.DPrint(true, "running database clean")
	out.DPrint(true, "list of buckets:")
	out.DPrint(true, strings.Join(buckets, "\n"))
}

func cleaner(db *bolt.DB, debug bool, buckets []string) ([]string, error) {
	if db == nil {
		return nil, bolt.ErrDatabaseNotOpen
	}
	if len(buckets) > 0 {
		return buckets, nil
	}

	out.DPrint(debug, "fetching all buckets")
	all, err := All(db)
	if err != nil {
		return nil, err
	}
	if len(all) == 0 {
		return nil, ErrEmpty
	}
	return all, nil
}

// Compact the database by reclaiming internal space.
func Compact(db *bolt.DB, debug bool) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	out.DPrint(debug, "running database compact")
	// active database
	src, err := DB()
	if err != nil {
		return err
	}
	// make a temporary database
	tmp := filepath.Join(os.TempDir(), backup())
	// open target database
	tmpDB, err := bolt.Open(tmp, PrivateFile, write())
	if err != nil {
		return err
	}
	out.DPrint(debug, "opened replacement database: "+tmp)
	defer tmpDB.Close()

	// compress and copy the results to the temporary database
	out.DPrint(debug, "compress and copy databases")
	if errComp := bolt.Compact(tmpDB, db, 0); errComp != nil {
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
		out.DPrint(debug, s1)
		s2 := fmt.Sprintf("new database:      %d bytes, %s", tm.Size(), tm.Name())
		out.DPrint(debug, s2)
	}
	if err = db.Close(); err != nil {
		return err
	}
	cp, err := CopyFile(tmp, src)
	if err != nil {
		return err
	}
	s := fmt.Sprintf("copied %d bytes to: %s", cp, src)
	out.DPrint(debug, s)
	return nil
}

// Compare finds exact matches of the string contained within the stored filenames and paths.
func Compare(db *bolt.DB, s string, buckets ...string) (*Matches, error) {
	const ignoreCase, pathBase = false, false
	return compare(db, ignoreCase, pathBase, []byte(s), buckets...)
}

// CompareBase finds exact matches of the string contained within the stored filenames.
func CompareBase(db *bolt.DB, s string, buckets ...string) (*Matches, error) {
	const ignoreCase, pathBase = false, true
	return compare(db, ignoreCase, pathBase, []byte(s), buckets...)
}

// CompareBaseNoCase finds case insensitive matches of the string contained within the stored filenames.
func CompareBaseNoCase(db *bolt.DB, s string, buckets ...string) (*Matches, error) {
	const ignoreCase, pathBase = true, true
	return compare(db, ignoreCase, pathBase, []byte(s), buckets...)
}

// CompareNoCase finds case insensitive matches of the string contained within the stored filenames and paths.
func CompareNoCase(db *bolt.DB, s string, buckets ...string) (*Matches, error) {
	const ignoreCase, pathBase = true, false
	return compare(db, ignoreCase, pathBase, []byte(s), buckets...)
}

func compare(db *bolt.DB, ignoreCase, pathBase bool, term []byte, buckets ...string) (*Matches, error) {
	if db == nil {
		return nil, bolt.ErrDatabaseNotOpen
	}
	if len(term) == 0 {
		return nil, ErrNoTerm
	}
	checked, err := checker(db, buckets)
	if err != nil {
		return nil, err
	}
	finds := make(Matches)
	for _, bucket := range checked {
		abs, err := AbsB(bucket)
		if err != nil {
			out.StderrCR(err)
		}
		if err = db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket(abs)
			if b == nil {
				return bolt.ErrBucketNotFound
			}
			s := term
			if ignoreCase {
				s = bytes.ToLower(term)
			}
			err = b.ForEach(func(key, _ []byte) error {
				k := compareKey(key, ignoreCase)
				if pathBase {
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
		}); errors.Is(err, bolt.ErrBucketNotFound) {
			return nil, fmt.Errorf("%w: '%s'", err, abs)
		} else if err != nil {
			return nil, err
		}
	}
	return &finds, nil
}

func checker(db *bolt.DB, buckets []string) ([]string, error) {
	if db == nil {
		return nil, bolt.ErrDatabaseNotOpen
	}
	if len(buckets) != 0 {
		return buckets, nil
	}
	all, err := All(db)
	if err != nil {
		return nil, err
	}
	if len(all) == 0 {
		return nil, ErrEmpty
	}
	return all, nil
}

func compareKey(key []byte, ignoreCase bool) []byte {
	k := key
	if ignoreCase {
		k = bytes.ToLower(key)
	}
	return k
}

// Count the number of records in the named bucket.
func Count(db *bolt.DB, name string) (int, error) {
	if db == nil {
		return 0, bolt.ErrDatabaseNotOpen
	}
	return bucket.Count(db, name)
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
	// create database directory if it doesn't exist
	if _, err = os.Stat(dir); os.IsNotExist(err) {
		if errMk := os.MkdirAll(dir, PrivateDir); errMk != nil {
			return "", fmt.Errorf("cannot create database directory: %w: %s", errMk, dir)
		}
	}
	// create a new database if it doesn't exist, this prevents
	// posix system returning the error a "bad file descriptor" when reading
	path := filepath.Join(dir, boltName)
	stat, errp := os.Stat(path)
	if os.IsNotExist(errp) {
		if err := Create(path); err != nil {
			return "", err
		}
		return path, nil
	}
	if errp != nil {
		return "", errp
	}
	// recreate an empty database if it is a zero byte file
	if stat.Size() == 0 {
		if err := os.Remove(path); err != nil {
			return "", err
		}
		if err := Create(path); err != nil {
			return "", err
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
func Info(db *bolt.DB) (string, error) {
	if db == nil {
		return "", bolt.ErrDatabaseNotOpen
	}
	path, err := DB()
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	w := new(tabwriter.Writer)
	w.Init(&b, 0, tabWidth, 0, '\t', 0)
	fmt.Fprintf(w, "\tLocation:\t%s", path)
	fmt.Fprintln(w)
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(w, "\n%s\n", NotFound)
			err = ErrNotFound
		}
		w.Flush()
		return b.String(), err
	}
	fmt.Fprintf(w, "\tModified:\t%s", stat.ModTime().Local().Format("Jan 2 15:04:05"))
	fmt.Fprintln(w)
	fmt.Fprintf(w, "\tFile:\t%s",
		color.Primary.Sprint(humanize.Bytes(uint64(stat.Size()))))
	if runtime.GOOS != winOS {
		fmt.Fprintf(w, " (%v)", stat.Mode())
	}
	fmt.Fprintln(w)
	var bucketsB int
	w, bucketsB, err = info(db, w, path)
	if err != nil {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "\tDatabase error:\t%s", err.Error())
		fmt.Fprintln(w)
	}
	const oneAndAHalf, oneMB = 1.5, 1_000_000
	tooBig := int64(float64(bucketsB) * oneAndAHalf)
	if stat.Size() > oneMB && stat.Size() > tooBig {
		fmt.Fprintln(w)
		fmt.Fprintln(w, color.Notice.Sprint("To reduce the size of the database:"))
		fmt.Fprintln(w, color.Debug.Sprint("dupers backup && dupers clean"))
	}
	w.Flush()
	return b.String(), nil
}

func info(db *bolt.DB, w *tabwriter.Writer, name string) (*tabwriter.Writer, int, error) {
	if db == nil {
		return nil, 0, bolt.ErrDatabaseNotOpen
	}
	type (
		vals struct {
			items int
			size  string
		}
		item map[string]vals
	)

	ro := color.Green.Sprint("OK")
	if !db.IsReadOnly() {
		ro = color.Danger.Sprint("NO")
	}
	fmt.Fprintf(w, "\tRead only mode:\t%s", ro)
	fmt.Fprintln(w)
	items, cnt, sizes := make(item), 0, 0
	if err := db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			v := tx.Bucket(name)
			if v == nil {
				return fmt.Errorf("%w: %s", bolt.ErrBucketNotFound, string(name))
			}
			cnt++
			sizes += v.Stats().LeafAlloc
			items[string(name)] = vals{v.Stats().KeyN, humanize.Bytes(uint64(v.Stats().LeafAlloc))}
			return nil
		})
	}); err != nil {
		return nil, 0, err
	}
	fmt.Fprintf(w, "Buckets:        %s", color.Primary.Sprint(cnt))
	fmt.Fprintln(w)
	fmt.Fprintln(w)
	if cnt == 0 {
		// exit when no buckets exist
		return w, sizes, nil
	}
	tab := tabwriter.NewWriter(w, 0, 0, tabPadding, ' ', tabwriter.AlignRight)
	fmt.Fprintf(tab, "Items\tSize\t\tBucket %s",
		color.Secondary.Sprint("(absolute path)"))
	fmt.Fprintln(tab)
	p := message.NewPrinter(language.English)
	for i, b := range items {
		p.Fprintf(tab, "%d\t%s\t\t%v", number.Decimal(b.items), b.size, i)
		fmt.Fprintln(tab)
	}
	if err := tab.Flush(); err != nil {
		return w, 0, err
	}
	return w, sizes, nil
}

// IsEmpty returns a bolt.ErrBucketNotFound error when the database has no buckets.
func IsEmpty(db *bolt.DB) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	cnt := 0
	if err := db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			cnt++
			return nil
		})
	}); err != nil {
		return err
	}
	if cnt == 0 {
		return bolt.ErrBucketNotFound
	}
	return nil
}

// List returns the filepaths and SHA256 checksums stored in the bucket.
func List(db *bolt.DB, bucket string) (Lists, error) {
	if db == nil {
		return nil, bolt.ErrDatabaseNotOpen
	}
	lists := make(Lists)
	if err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return bolt.ErrBucketNotFound
		}
		h := [32]byte{}
		return b.ForEach(func(k, v []byte) error {
			copy(h[:], v)
			lists[Filepath(k)] = h
			return nil
		})
	}); err != nil {
		return nil, err
	}
	return lists, nil
}

// Rename the named bucket in the database to use a new, target directory path.
func Rename(db *bolt.DB, name, target string) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	if name == target {
		return ErrSameNames
	}
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(name))
		if b == nil {
			return bolt.ErrBucketNotFound
		}
		ren, errRen := tx.CreateBucket([]byte(target))
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
func RM(db *bolt.DB, name string) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	return db.Update(func(tx *bolt.Tx) error {
		if b := tx.Bucket([]byte(name)); b == nil {
			return bolt.ErrBucketNotFound
		}
		return tx.DeleteBucket([]byte(name))
	})
}
