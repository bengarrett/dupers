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

	"github.com/bengarrett/dupers/internal/printer"
	"github.com/bengarrett/dupers/pkg/database/bucket"
	"github.com/dustin/go-humanize"
	"github.com/gookit/color"
	bolt "go.etcd.io/bbolt"
	bberr "go.etcd.io/bbolt/errors"
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
	PrivateFile fs.FileMode = 0o600 // PrivateFile mode means only the owner has read/write access.
	PrivateDir  fs.FileMode = 0o700 // PrivateDir mode means only the owner has read/write/dir access.

	NotFound = "This is okay as one will be created when using the dupe or search commands."

	backupTime = "20060102-150405"
	boltName   = "dupers.db"
	csvName    = "dupers-export.csv"
	subdir     = "dupers"
	winOS      = "windows"
	tabPadding = 4
	tabWidth   = 8
)

var (
	ErrEmpty     = errors.New("database is empty and contains no items")
	ErrNoCompact = errors.New("compression has not reduced the database size")
	ErrNoClean   = errors.New("database has nothing to clean")
	ErrNoTerm    = errors.New("cannot compare an empty term")
	ErrNotFound  = errors.New("database file does not exist")
	ErrSameName  = errors.New("bucket target is the same as the bucket name")
	ErrZeroByte  = errors.New("database is a zero byte file and is unusable")
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
		return nil, bberr.ErrDatabaseNotOpen
	}
	var names []string
	if err := db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			if v := tx.Bucket(name); v == nil {
				return fmt.Errorf("%w: %s", bberr.ErrBucketNotFound, string(name))
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
		printer.StderrCR(ErrNotFound)
		fmt.Fprintf(w, "\n%s\nThe database will be located at: %s\n", NotFound, path)
		return 0, ErrNotFound // 0
	} else if err1 != nil {
		return 0, err1
	}
	if i.Size() == 0 {
		printer.StderrCR(ErrZeroByte)
		s := "This error occures when dupers cannot save any data to the file system."
		fmt.Fprintf(w, "\n%s\nThe database is located at: %s\n", s, path)
		return 0, ErrZeroByte // 1
	}
	return i.Size(), nil
}

// Exist returns an error if the bucket does not exists in the database.
func Exist(db *bolt.DB, bucket string) error {
	if db == nil {
		return bberr.ErrDatabaseNotOpen
	}
	return db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return bberr.ErrBucketNotFound
		}
		return nil
	})
}

// Clean the stale items from database buckets.
// Stale items are file pointers that no longer exist on the host file system.
func Clean(db *bolt.DB, quiet, debug bool, buckets ...string) error {
	if db == nil {
		return bberr.ErrDatabaseNotOpen
	}
	cleanDebug(debug, buckets)

	printer.Debug(debug, fmt.Sprintf("cleaner of buckets: %s", buckets))
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
		return ErrNoClean
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
	printer.Debug(true, "running database clean")
	printer.Debug(true, "list of buckets:")
	printer.Debug(true, strings.Join(buckets, "\n"))
}

func cleaner(db *bolt.DB, debug bool, buckets []string) ([]string, error) {
	if db == nil {
		return nil, bberr.ErrDatabaseNotOpen
	}
	if len(buckets) > 0 {
		return buckets, nil
	}

	printer.Debug(debug, "fetching all buckets")
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
		return bberr.ErrDatabaseNotOpen
	}
	printer.Debug(debug, "running database compact")
	// make a temporary database
	f, err := os.CreateTemp(os.TempDir(), "dupers-*.db")
	if err != nil {
		return err
	}
	defer f.Close()

	// open target database
	target, err := bolt.Open(f.Name(), PrivateFile, write())
	if err != nil {
		return fmt.Errorf("%w: open %s", err, f.Name())
	}
	printer.Debug(debug, "opened replacement database: "+f.Name())
	defer target.Close()

	// compress and copy the results to the temporary database
	printer.Debug(debug, "compress and copy databases")
	if err := bolt.Compact(target, db, 0); err != nil {
		return fmt.Errorf("%w: compact %s", err, f.Name())
	}
	if debug {
		statSrc, err := os.Stat(db.Path())
		if err != nil {
			return err
		}
		statDst, err := os.Stat(f.Name())
		if err != nil {
			return err
		}
		s1 := fmt.Sprintf("original database: %d bytes, %s", statSrc.Size(), statSrc.Name())
		printer.Debug(debug, s1)
		s2 := fmt.Sprintf("new database:      %d bytes, %s", statDst.Size(), statDst.Name())
		printer.Debug(debug, s2)
	}
	path := db.Path()
	if err = db.Close(); err != nil {
		return err
	}
	i, err := CopyFile(f.Name(), path)
	if err != nil {
		return err
	}
	s := fmt.Sprintf("copied %d bytes to: %s", i, path)
	printer.Debug(debug, s)
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
		return nil, bberr.ErrDatabaseNotOpen
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
			printer.StderrCR(err)
		}
		err = db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket(abs)
			if b == nil {
				return bberr.ErrBucketNotFound
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
		})
		if err != nil {
			if errors.Is(err, bberr.ErrBucketNotFound) {
				return nil, fmt.Errorf("%w: '%s'", err, abs)
			}
			return nil, err
		}
	}
	return &finds, nil
}

func checker(db *bolt.DB, buckets []string) ([]string, error) {
	if db == nil {
		return nil, bberr.ErrDatabaseNotOpen
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
		return 0, bberr.ErrDatabaseNotOpen
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
		return "", bberr.ErrDatabaseNotOpen
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
		color.Primary.Sprint(humanize.Bytes(safesize(stat.Size()))))
	if runtime.GOOS != winOS {
		fmt.Fprintf(w, " (%v)", stat.Mode())
	}
	fmt.Fprintln(w)
	var bucketsB int
	w, bucketsB, err = info(db, w)
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

func safesize(i int64) uint64 {
	if i < 0 {
		return 0
	}
	return uint64(i)
}

func info(db *bolt.DB, w *tabwriter.Writer) (*tabwriter.Writer, int, error) {
	if db == nil {
		return nil, 0, bberr.ErrDatabaseNotOpen
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
				return fmt.Errorf("%w: %s", bberr.ErrBucketNotFound, string(name))
			}
			cnt++
			sizes += v.Stats().LeafAlloc
			s := int64(v.Stats().LeafAlloc)
			items[string(name)] = vals{v.Stats().KeyN, humanize.Bytes(safesize(s))}
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

// IsEmpty returns a bberr.ErrBucketNotFound error when the database has no buckets.
func IsEmpty(db *bolt.DB) error {
	if db == nil {
		return bberr.ErrDatabaseNotOpen
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
		return bberr.ErrBucketNotFound
	}
	return nil
}

// List returns the filepaths and SHA256 checksums stored in the bucket.
func List(db *bolt.DB, bucket string) (Lists, error) {
	if db == nil {
		return nil, bberr.ErrDatabaseNotOpen
	}
	lists := make(Lists)
	if err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return bberr.ErrBucketNotFound
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
		return bberr.ErrDatabaseNotOpen
	}
	if name == target {
		return ErrSameName
	}
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(name))
		if b == nil {
			return bberr.ErrBucketNotFound
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

// Remove the named bucket from the database.
func Remove(db *bolt.DB, name string) error {
	if db == nil {
		return bberr.ErrDatabaseNotOpen
	}
	return db.Update(func(tx *bolt.Tx) error {
		if b := tx.Bucket([]byte(name)); b == nil {
			return bberr.ErrBucketNotFound
		}
		return tx.DeleteBucket([]byte(name))
	})
}
