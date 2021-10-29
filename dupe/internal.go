// © Ben Garrett https://github.com/bengarrett/dupers

// Package dupers is the blazing-fast file duplicate checker and filename search.
package dupe

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bengarrett/dupers/database"
	"github.com/bengarrett/dupers/out"
	"github.com/gookit/color"
	bolt "go.etcd.io/bbolt"
)

type internal struct {
	db      *bolt.DB  // Bolt database
	buckets []Bucket  // buckets to lookup
	compare checksums // hashes fetched from the database or file system
	files   int       // nolint: structcheck
	sources []string  // nolint: structcheck
	source  string    // directory or file to compare
	timer   time.Time
}

// OpenRead opens the Bolt database for reading.
func (i *internal) OpenRead() {
	if i.db != nil {
		return
	}
	db, err := database.OpenRead()
	if err != nil {
		out.ErrFatal(err)
	}
	i.db = db
}

// OpenWrite opens the Bolt database for reading and writing.
func (i *internal) OpenWrite() {
	if i.db != nil {
		return
	}
	db, err := database.OpenWrite()
	if err != nil {
		out.ErrFatal(err)
	}
	i.db = db
}

// SetAllBuckets sets all the database backets for use with the dupe or search.
func (i *internal) SetAllBuckets() {
	names, err := database.AllBuckets(nil)
	if err != nil {
		out.ErrFatal(err)
	}
	for _, name := range names {
		i.buckets = append(i.buckets, Bucket(name))
	}
}

// SetBuckets adds the bucket name to a list of buckets.
func (i *internal) SetBuckets(names ...string) {
	for _, name := range names {
		i.buckets = append(i.buckets, Bucket(name))
	}
}

// SetCompares fetches items from the named bucket and sets them to c.compare.
func (i *internal) SetCompares(name Bucket) int {
	ls, err := database.List(string(name), i.db)
	if err != nil {
		out.ErrCont(err)
	}
	if i.compare == nil {
		i.compare = make(checksums)
	}
	for fp, sum := range ls {
		i.compare[sum] = string(fp)
	}
	return len(ls)
}

// Compares returns the number of items contained in c.compare.
func (i *internal) Compares() int {
	return len(i.compare)
}

// SetTimer starts a process timer.
func (i *internal) SetTimer() {
	i.timer = time.Now()
}

// SetToCheck sets the named string as the directory or file to check.
func (i *internal) SetToCheck(name string) {
	n, err := filepath.Abs(name)
	if err != nil {
		out.ErrFatal(err)
	}
	i.source = n
}

// Buckets returns a slice of Buckets.
func (i *internal) Buckets() []Bucket {
	return i.buckets
}

// PrintBuckets returns a list of buckets used by the database.
func (i *internal) PrintBuckets() string {
	s := make([]string, len(i.Buckets()))
	for _, b := range i.Buckets() {
		s = append(s, string(b))
	}
	return strings.Join(s, " ")
}

// ToCheck returns the directory or file to check.
func (i *internal) ToCheck() string {
	return i.source
}

// Timer returns the time taken since the process timer was instigated.
func (i *internal) Timer() time.Duration {
	return time.Since(i.timer)
}

// Print the results of the database comparisons.
func Print(quiet bool, m *database.Matches) string {
	if m == nil || len(*m) == 0 {
		return ""
	}
	w := new(bytes.Buffer)
	// collect the bucket names which will be used to sort the results
	buckets, bucket := []string{}, ""
	for _, bucket := range *m {
		if !contains(string(bucket), buckets...) {
			buckets = append(buckets, string(bucket))
		}
	}
	sort.Strings(buckets)
	for i, buck := range buckets {
		cnt := 0
		if i > 0 {
			fmt.Fprintln(w)
		}
		// print the matches, the filenames are unsorted
		for file, b := range *m {
			if string(b) != buck {
				continue
			}
			cnt++
			if string(b) != bucket {
				bucket = string(b)
				if !quiet {
					if cnt > 1 {
						fmt.Fprintln(w)
					}
					fmt.Fprintf(w, "%s: %s\n", color.Info.Sprint("Results from"), b)
				}
			}
			if quiet {
				fmt.Fprintf(w, "%s\n", file)
				continue
			}
			if cnt == 1 {
				fmt.Fprintf(w, "%s%s\n", color.Success.Sprint("  ⤷\t"), file)
				continue
			}
			fmt.Fprintf(w, "  %s%s\t%s\n", color.Primary.Sprint(cnt), color.Secondary.Sprint("."), file)
		}
	}
	return w.String()
}

// contains returns true if find exists in s.
func contains(find string, s ...string) bool {
	for _, item := range s {
		if find == item {
			return true
		}
	}
	return false
}
