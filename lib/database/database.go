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

type matches map[string]string

const (
	FileMode fs.FileMode = 0600
	dbName               = "dupers.db"
	dbPath               = "dupers"
)

var ErrNoBucket = errors.New("bucket does not exist")

func Backup() (name string, written int64, err error) {
	src, err := DB()
	if err != nil {
		return "", 0, err
	}
	now := time.Now().Format("20060102-150405")
	ext := filepath.Ext(dbName)
	file := fmt.Sprintf("%s-backup-%s%s", strings.TrimSuffix(dbName, ext), now, ext)

	dir, err := os.UserHomeDir()
	if err != nil {
		return "", 0, err
	}
	name = filepath.Join(dir, file)

	written, err = copy(src, name)
	if err != nil {
		return "", 0, err
	}
	return name, written, nil
}

func copy(src, dest string) (int64, error) {
	// read source
	f, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	// create backup file
	new, err := os.Create(dest)
	if err != nil {
		return 0, err
	}
	defer new.Close()
	// duplicate data
	return io.Copy(new, f)
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

	if err = db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			v := tx.Bucket(name)
			if v == nil {
				return fmt.Errorf("%w: %s", ErrNoBucket, string(name))
			}
			names = append(names, string(name))
			return nil
		})
	}); err != nil {
		return nil, err
	}
	return names, nil
}

// Clean the stale items from all database buckets.
// Stale items are file pointers that no longer exist on the host file system.
func Clean() error {
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
				if _, err := os.Stat(string(k)); err != nil {
					if err = db.Update(func(tx *bolt.Tx) error {
						return tx.Bucket(abs).Delete(k)
					}); err != nil {
						return err
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
	if cnt == 0 {
		fmt.Println("nothing was cleaned")
		return nil
	}
	fmt.Println("removed", cnt, "stale items")
	return nil
}

// Compare finds exact matches of the string contained within the stored filenames and paths.
func Compare(s string, buckets []string) error {
	return compare([]byte(s), buckets, false, false)
}

// CompareBase finds exact matches of the string contained within the stored filenames.
func CompareBase(s string, buckets []string) error {
	return compare([]byte(s), buckets, false, true)
}

// CompareBaseNoCase finds case insensitive matches of the string contained within the stored filenames.
func CompareBaseNoCase(s string, buckets []string) error {
	return compare([]byte(s), buckets, true, true)
}

// CompareNoCase finds case insensitive matches of the string contained within the stored filenames and paths.
func CompareNoCase(s string, buckets []string) error {
	return compare([]byte(s), buckets, true, false)
}

func bucketAbs(name string) ([]byte, error) {
	s, err := filepath.Abs(name)
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
}

func compare(term []byte, buckets []string, noCase, base bool) error {
	path, err := DB()
	if err != nil {
		log.Fatalln(err)
	}
	db, err := bolt.Open(path, FileMode, &bolt.Options{ReadOnly: true})
	if err != nil {
		return err
	}
	defer db.Close()

	if len(buckets) == 0 {
		buckets, err = Buckets()
		if err != nil {
			return err
		}
	}

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
			b.ForEach(func(key, _ []byte) error {
				k := key
				if noCase {
					k = bytes.ToLower(key)
				}
				if base {
					k = []byte(filepath.Base(string(k)))
					if bytes.Contains(k, s) {
						fmt.Printf("\nMatch: %s\n", key)
					}
					return nil
				}
				if bytes.Contains(k, s) {
					fmt.Printf("\nMatch: %s\n", key)
				}
				return nil
			})
			return nil
		}); err != nil {
			log.Println(err)
		}
	}
	return nil
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

// RM removes the named bucket from the database.
func RM(name string) error {
	path, err := DB()
	if err != nil {
		return err
	}
	db, err := bolt.Open(path, FileMode, &bolt.Options{})
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
		h := []byte(hash[:])
		b.ForEach(func(k, v []byte) error {
			records++
			if bytes.Equal(v, h) {
				finds = append(finds, string(k))
			}
			return nil
		})
		return nil
	})
}
