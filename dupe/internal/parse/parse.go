// © Ben Garrett https://github.com/bengarrett/dupers
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
	"sort"
	"strings"
	"time"

	"github.com/bengarrett/dupers/database"
	"github.com/bengarrett/dupers/internal/out"
	"github.com/gookit/color"
	bolt "go.etcd.io/bbolt"
)

type (
	Bucket    string
	Checksum  [32]byte
	Checksums map[Checksum]string
)

const (
	oneKb = 1024
	oneMb = oneKb * oneKb
)

type Parser struct {
	DB      *bolt.DB  // Bolt database.
	Buckets []Bucket  // Buckets to lookup.
	Compare Checksums // Compare hashes fetched from the database or file system.
	Files   int       // Files count.
	Sources []string  // Sources are directories or files to compare.
	Source  string    // Source directory or file to compare.
	timer   time.Time
}

// All buckets returned as a slice.
func (p *Parser) All() []Bucket {
	return p.Buckets
}

// Compares the number of items contained in c.compare.
func (p *Parser) Compares() int {
	return len(p.Compare)
}

// OpenRead opens the Bolt database for reading.
// func (p *Parser) OpenRead() {
// 	if p.DB != nil {
// 		return
// 	}
// 	db, err := database.OpenRead()
// 	if err != nil {
// 		out.ErrFatal(err)
// 	}
// 	p.DB = db
// }

// OpenWrite opens the Bolt database for reading and writing.
// func (p *Parser) OpenWrite() {
// 	if p.DB != nil {
// 		return
// 	}
// 	db, err := database.OpenWrite()
// 	if err != nil {
// 		out.ErrFatal(err)
// 	}
// 	p.DB = db
// }

// SetAllBuckets sets all the database buckets for use with the dupe or search commands.
func (p *Parser) SetAllBuckets(db *bolt.DB) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
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
func (p *Parser) SetBuckets(names ...string) error {
	var errs error
	for _, name := range names {
		n, err := filepath.Abs(name)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("%w: %s", err, n))
			continue
		}
		if _, err := os.Stat(n); err != nil {
			errs = errors.Join(errs, fmt.Errorf("%w: %s", err, n))
			continue
		}

		p.Buckets = append(p.Buckets, Bucket(name))
	}
	if errs != nil {
		return errs
	}
	return nil
}

// SetCompares fetches items from the named bucket and sets them to p.Compare.
func (p *Parser) SetCompares(db *bolt.DB, name Bucket) (int, error) {
	if db == nil {
		return 0, bolt.ErrDatabaseNotOpen
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
func (p *Parser) SetTimer() {
	p.timer = time.Now()
}

// SetSource sets the named string as the directory or file to check.
func (p *Parser) SetSource(name string) error {
	n, err := filepath.Abs(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(n); err != nil {
		return err
	}
	p.Source = n
	return nil
}

// PrintBuckets returns a list of buckets used by the database.
func (p *Parser) PrintBuckets() string {
	s := make([]string, 0, len(p.All()))
	for _, b := range p.All() {
		s = append(s, string(b))
	}
	return strings.Join(s, " ")
}

// ToCheck returns the directory or file to check.
func (p *Parser) ToCheck() string {
	return p.Source
}

// Timer returns the time taken since the process timer was instigated.
func (p *Parser) Timer() time.Duration {
	return time.Since(p.timer)
}

// Contains returns true if find exists in s.
func Contains(find string, s ...string) bool {
	for _, item := range s {
		if find == item {
			return true
		}
	}
	return false
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
				fmt.Fprintf(w, "%s%s\n", color.Success.Sprint(out.MatchPrefix),
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
	f, err := os.Open(name)
	if err != nil {
		return Checksum{}, err
	}
	defer f.Close()

	buf, h := make([]byte, oneMb), sha256.New()
	if _, err := io.CopyBuffer(h, f, buf); err != nil {
		return Checksum{}, err
	}
	var c Checksum
	copy(c[:], h.Sum(nil))
	return c, nil
}

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
func Executable(root string) bool {
	bin := false
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if bin {
			return nil
		}
		if !d.IsDir() {
			if program(d.Name()) {
				bin = true
				return nil
			}
		}
		return nil
	}); err != nil {
		out.ErrCont(err)
	}
	return bin
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
