// © Ben Garrett https://github.com/bengarrett/dupers

// dupers todo
package database

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/dustin/go-humanize"
	bolt "go.etcd.io/bbolt"
)

const (
	dbName = "dupers.db"
	dbPath = "dupers"
)

var ErrNoBucket = errors.New("bucket does not exist")

func Info() string {
	n, err := Name()
	if err != nil {
		log.Fatalln(err)
	}
	var b bytes.Buffer
	w := new(tabwriter.Writer)
	w.Init(&b, 0, 8, 0, '\t', 0)
	fmt.Fprintf(w, "\tLocation:\t%s\n", n)
	s, err := os.Stat(n)
	if err != nil {
		fmt.Fprintln(w, "\t\tThe database doesn't exist, but one will be created during the next scan")
		w.Flush()
		return b.String()
	}
	fmt.Fprintf(w, "\tFile size:\t%s\n", humanize.Bytes(uint64(s.Size())))
	w, err = boltInf(n, w)
	if err != nil {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "\tDatabase error:\t%s\n", err.Error())
	}
	w.Flush()
	return b.String()
}

func boltInf(name string, w *tabwriter.Writer) (*tabwriter.Writer, error) {
	db, err := bolt.Open(name, 0600, &bolt.Options{ReadOnly: true})
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
			fmt.Fprintf(w, "\t\titems: %d\tsize: %s\thashes: %s\n", v.Stats().KeyN, humanize.Bytes(uint64(tx.Size())), humanize.Bytes(uint64(v.Stats().LeafInuse)))
			return nil
		})
	})
	return w, err
}

func Name() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir, err = os.UserHomeDir()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(dir, dbPath, dbName), nil
}
