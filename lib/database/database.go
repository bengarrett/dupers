// © Ben Garrett https://github.com/bengarrett/dupers

// dupers todo
package database

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/dustin/go-humanize"
	bolt "go.etcd.io/bbolt"
)

type (
	Filepath string
	Matches  map[Filepath]string
)

const (
	FileMode fs.FileMode = 0600
	dbName               = "dupers.db"
	dbPath               = "dupers"
)

var (
	ErrNoBucket  = errors.New("bucket does not exist")
	ErrDBClean   = errors.New("database had nothing to clean")
	ErrDBCompact = errors.New("database compression has not reduced the size")
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
	now, ext := time.Now().Format("20060102-150405"), filepath.Ext(dbName)
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

// Buckets lists all the stored bucket names in the database.
func Buckets() (names []string, err error) {
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
func Clean(quiet bool) error {
	buckets, err := Buckets()
	if err != nil {
		return err
	}
	path, err := DB()
	if err != nil {
		log.Fatalln(err)
	}
	db, err := bolt.Open(path, FileMode, nil)
	if err != nil {
		return err
	}
	defer db.Close()
	cnt := 0
	for _, bucket := range buckets {
		abs, err := bucketAbs(bucket)
		if err != nil {
			log.Println(err)
		}
		if err = db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket(abs)
			if b == nil {
				return ErrNoBucket
			}
			err = b.ForEach(func(k, v []byte) error {
				if _, err1 := os.Stat(string(k)); err1 != nil {
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
				log.Println(err)
			}
			return nil
		}); err != nil {
			log.Println(err)
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
func Compact() error {
	// active database
	src, err := DB()
	if err != nil {
		return err
	}
	// make a temporary database
	tmp := filepath.Join(os.TempDir(), backupName())
	// open both databases
	srcDB, err := bolt.Open(src, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer srcDB.Close()
	tmpDB, err := bolt.Open(tmp, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer tmpDB.Close()
	// compress and copy the results to the temporary database
	if err = bolt.Compact(tmpDB, srcDB, 0); err != nil {
		log.Fatalln(err)
	}
	srcSt, err := os.Stat(src)
	if err != nil {
		log.Fatalln(err)
	}
	tmpSt, err := os.Stat(tmp)
	if err != nil {
		log.Fatalln(err)
	}
	// compare size of the two databases
	// if the compacted temporary database is smaller,
	// copy it to the active database
	if tmpSt.Size() >= srcSt.Size() {
		tmpDB.Close()
		if err := os.Remove(tmp); err != nil {
			log.Println(err)
		}
		return ErrDBCompact
	}
	srcDB.Close()
	_, err = copyFile(tmp, src)
	if err != nil {
		return err
	}
	return nil
}

// Compare finds exact matches of the string contained within the stored filenames and paths.
func Compare(s string, buckets []string) (*Matches, error) {
	return compare([]byte(s), buckets, false, false)
}

// CompareBase finds exact matches of the string contained within the stored filenames.
func CompareBase(s string, buckets []string) (*Matches, error) {
	return compare([]byte(s), buckets, false, true)
}

// CompareBaseNoCase finds case insensitive matches of the string contained within the stored filenames.
func CompareBaseNoCase(s string, buckets []string) (*Matches, error) {
	m, err := compare([]byte(s), buckets, true, true)
	return m, err
}

// CompareNoCase finds case insensitive matches of the string contained within the stored filenames and paths.
func CompareNoCase(s string, buckets []string) (*Matches, error) {
	return compare([]byte(s), buckets, true, false)
}

func bucketAbs(name string) ([]byte, error) {
	s, err := filepath.Abs(name)
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
}

func compare(term []byte, buckets []string, noCase, base bool) (*Matches, error) {
	path, err := DB()
	if err != nil {
		log.Fatalln(err)
	}
	db, err := bolt.Open(path, FileMode, &bolt.Options{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if len(buckets) == 0 {
		buckets, err = Buckets()
		if err != nil {
			return nil, err
		}
	}

	finds := make(Matches)
	for _, bucket := range buckets {
		abs, err := bucketAbs(bucket)
		if err != nil {
			log.Println(err)
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
						finds[Filepath(key)] = bucket
					}
					return nil
				}
				if bytes.Contains(k, s) {
					finds[Filepath(key)] = bucket
				}
				return nil
			})
			return err
		}); errors.Is(err, ErrNoBucket) {
			return nil, fmt.Errorf("%w: %q", err, abs)
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
	return filepath.Join(dir, dbPath, dbName), nil
}

// Info returns a printout of the buckets and their statistics.
func Info() string {
	path, err := DB()
	if err != nil {
		log.Fatalln(err)
	}
	var b bytes.Buffer
	w := new(tabwriter.Writer)
	w.Init(&b, 0, 8, 0, '\t', 0)
	fmt.Fprintf(w, "\tLocation:\t%s\n", path)
	s, err := os.Stat(path)
	if err != nil {
		fmt.Fprintln(w, "\t\tThe database doesn't exist, but one will be created during the next scan")
		w.Flush()
		return b.String()
	}
	fmt.Fprintf(w, "\tFile size:\t%s\n", humanize.Bytes(uint64(s.Size())))
	w, err = info(path, w)
	if err != nil {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "\tDatabase error:\t%s\n", err.Error())
	}
	w.Flush()
	return b.String()
}

func info(name string, w *tabwriter.Writer) (*tabwriter.Writer, error) {
	db, err := bolt.Open(name, FileMode, &bolt.Options{ReadOnly: true})
	if err != nil {
		return w, err
	}
	defer db.Close()
	fmt.Fprintf(w, "\tRead only mode:\t%v\n", db.IsReadOnly())
	err = db.View(func(tx *bolt.Tx) error {
		cnt := 0
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			v := tx.Bucket(name)
			if v == nil {
				return fmt.Errorf("%w: %s", ErrNoBucket, string(name))
			}
			cnt++
			fmt.Fprintln(w)
			fmt.Fprintf(w, "\tBucket #%002d\t%q\n", cnt, string(name))
			fmt.Fprintf(w, "\t\titems: %d\tdata: %s\n", v.Stats().KeyN, humanize.Bytes(uint64(v.Stats().LeafInuse)))
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
	if err = db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			cnt++
			return nil
		})
	}); err != nil {
		return true, err
	}
	if cnt == 0 {
		return true, nil
	}
	return false, nil
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
		return tx.DeleteBucket([]byte(name))
	})
}

// Seek searches a bucket for an exact SHA256 hash match.
func Seek(hash [32]byte, bucket string) (finds []string, records int, err error) {
	path, err := DB()
	if err != nil {
		log.Fatalln(err)
	}
	db, err := bolt.Open(path, FileMode, &bolt.Options{ReadOnly: true})
	if err != nil {
		return nil, records, err
	}
	defer db.Close()
	return finds, records, db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrNoBucket
		}
		h := hash[:]
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
