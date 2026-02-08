// © Ben Garrett https://github.com/bengarrett/dupers

// Package parse provides file parsing and checksum calculation functionality.
package parse

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/bengarrett/dupers/internal/printer"
	"github.com/bengarrett/dupers/pkg/database"
	"github.com/gookit/color"
	bolt "go.etcd.io/bbolt"
	bberr "go.etcd.io/bbolt/errors"
)

type (
	Bucket    string              // Bucket is a database table named as an absolute directory path.
	Checksum  [32]byte            // Checksum is a SHA-1 hash file value.
	Checksums map[Checksum]string // Checksums is a collection of SHA-1 hash file values.
)

const (
	oneKb = 1024
	oneMb = oneKb * oneKb
)

type Scanner struct {
	Sources []string  // Sources to compare, either directories or files.
	Buckets []Bucket  // Buckets to lookup.
	Compare Checksums // Compare hashes fetched from the database or file system.
	Files   int       // Files counter of the totals scanned and processed.
	timer   time.Time
}

var ErrNoSource = errors.New("cannot use an empty source")

// Compares returns the number of items in the Compare Scanner.
func (p *Scanner) Compares() int {
	return len(p.Compare)
}

// SetAllBuckets sets all the database buckets for use with the dupe or search commands.
func (p *Scanner) SetAllBuckets(db *bolt.DB) error {
	if db == nil {
		return bberr.ErrDatabaseNotOpen
	}
	names, err := database.All(db)
	if err != nil {
		return err
	}
	for _, name := range names {
		p.Buckets = append(p.Buckets, Bucket(name))
	}
	return nil
}

// SetBuckets adds the bucket name to a list of buckets.
func (p *Scanner) SetBuckets(names ...string) error {
	// find returns true if n exists in p.Buckets
	find := func(n string) bool {
		for _, x := range p.Buckets {
			if n == string(x) {
				return true
			}
		}
		return false
	}

	var errs error
	for _, name := range names {
		if name == "" {
			continue
		}
		n, err := filepath.Abs(name)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("%w: %s", err, n))
			continue
		}
		if _, err := os.Stat(n); err != nil {
			errs = errors.Join(errs, fmt.Errorf("%w: %s", err, n))
			continue
		}
		if find(n) {
			continue
		}
		p.Buckets = append(p.Buckets, Bucket(n))
	}
	if errs != nil {
		return errs
	}
	return nil
}

// SetCompares fetches item names an checksums from the named bucket and stores them in the Compare Scanner.
func (p *Scanner) SetCompares(db *bolt.DB, name Bucket) (int, error) {
	if db == nil {
		return 0, bberr.ErrDatabaseNotOpen
	}
	ls, err := database.List(db, string(name))
	if err != nil {
		return 0, err
	}
	if p.Compare == nil {
		p.Compare = make(Checksums)
	}
	for fp, sum := range ls {
		p.Compare[sum] = string(fp)
	}
	return len(ls), nil
}

// SetTimer starts a process timer.
func (p *Scanner) SetTimer() {
	p.timer = time.Now()
}

// GetSource returns the directory or file to check.
func (p *Scanner) GetSource() string {
	if len(p.Sources) == 0 {
		return ""
	}
	return p.Sources[0]
}

// SetSource sets the named string as the directory or file to check.
func (p *Scanner) SetSource(name string) error {
	if name == "" {
		return ErrNoSource
	}
	n, err := filepath.Abs(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(n); err != nil {
		return err
	}
	if len(p.Sources) == 0 {
		p.Sources = append(p.Sources, n)
		return nil
	}
	p.Sources[0] = n
	return nil
}

// BucketS returns a list of buckets used by the database.
func (p *Scanner) BucketS() string {
	s := make([]string, 0, len(p.Buckets))
	for _, b := range p.Buckets {
		s = append(s, string(b))
	}
	return strings.Join(s, " ")
}

// Timer returns the time taken since the process timer was instigated.
func (p *Scanner) Timer() time.Duration {
	return time.Since(p.timer)
}

// Contains returns true if find exists in s.
func Contains(find string, s ...string) bool {
	return slices.Contains(s, find)
}

// Print the results of the database comparisons.
func Print(quiet, exact bool, term string, m *database.Matches) string {
	if m == nil || len(*m) == 0 {
		return ""
	}
	w := new(bytes.Buffer)
	// collect the bucket names which will be used to sort the results
	bucket, buckets := matchBuckets(m)
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
					fmt.Fprintf(w, "%s: %s", color.Info.Sprint("Search results in"), b)
				}
			}
			if quiet {
				fmt.Fprintf(w, "%s\n", file)
				continue
			}
			mark := Marker(file, term, exact)
			if cnt == 1 {
				fmt.Fprintf(w, "%s%s\n", color.Success.Sprint(printer.MatchPrefix),
					mark)
				continue
			}
			fmt.Fprintf(w, "  %s%s\t%s\n", color.Primary.Sprint(cnt),
				color.Secondary.Sprint("."), mark)
		}
	}
	return w.String()
}

// Read the named file to return a SHA256 checksum of it's data.
func Read(name string) (Checksum, error) {
	name = filepath.Clean(name)
	f, err := os.Open(name)
	if err != nil {
		return Checksum{}, err
	}
	defer func() { _ = f.Close() }()
	buf, h := make([]byte, oneMb), sha256.New()
	if _, err := io.CopyBuffer(h, f, buf); err != nil {
		return Checksum{}, err
	}
	var c Checksum
	copy(c[:], h.Sum(nil))
	return c, nil
}

// Marker uses ANSI color to highlight the term contained in the filepath.
func Marker(file database.Filepath, term string, exact bool) string {
	s := string(file)
	switch {
	case !color.Enable:
		return s
	case !exact:
		return markInsensitive(s, term)
	default:
		return markExact(s, term)
	}
}

func markExact(s, substr string) string {
	return strings.ReplaceAll(s, substr, color.Info.Sprint(substr))
}

func markInsensitive(s, substr string) string {
	re := regexp.MustCompile(fmt.Sprintf("(?i)(%s)", substr))
	return re.ReplaceAllString(s, color.Info.Sprint("$1"))
}

func matchBuckets(m *database.Matches) (string, []string) {
	bucket, buckets := "", []string{}
	for _, bucket := range *m {
		if !Contains(string(bucket), buckets...) {
			buckets = append(buckets, string(bucket))
		}
	}
	sort.Strings(buckets)
	return bucket, buckets
}

// Executable returns true if the root directory contains an MS-DOS or Windows program file.
func Executable(root string) (bool, error) {
	isProgram := false
	if err := filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if isProgram {
			return nil
		}
		if !d.IsDir() {
			if program(d.Name()) {
				isProgram = true
				return nil
			}
		}
		return nil
	}); err != nil {
		return false, err
	}
	return isProgram, nil
}

// program returns true if the named file uses an MS-DOS or Windows program file extension.
func program(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".com", ".exe":
		return true
	default:
		return false
	}
}
