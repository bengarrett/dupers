// © Ben Garrett https://github.com/bengarrett/dupers

// Package dupers is the blazing-fast file duplicate checker and filename search.
package dupers

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bengarrett/dupers/database"
	"github.com/bengarrett/dupers/out"
	"github.com/dustin/go-humanize"
	"github.com/gookit/color"
	"github.com/karrick/godirwalk"
	bolt "go.etcd.io/bbolt"
)

type (
	Bucket    string
	checksum  [32]byte
	checksums map[checksum]string
)

const (
	modFmt = "02 Jan 2006 15:04"
	winOS  = "windows"
)

// Config options for duper.
type Config struct {
	Debug bool // spam the feedback sent to stdout
	Quiet bool // reduce the feedback sent to stdout
	Test  bool // internal unit test mode
	internal
}

type internal struct {
	db      *bolt.DB  // Bolt database
	buckets []Bucket  // buckets to lookup
	compare checksums // hashes fetched from the database or file system
	files   int       // total files or database items read
	sources []string  // files paths to check
	source  string    // directory or file to compare
	timer   time.Time
}

// SetAllBuckets sets all the database backets for use with the dupe or search.
func (i *internal) SetAllBuckets() {
	names, err := database.AllBuckets()
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
func (i *internal) SetCompares(name Bucket) {
	ls, err := database.List(string(name))
	if err != nil {
		out.ErrCont(err)
	}
	for fp, sum := range ls {
		i.compare[sum] = string(fp)
	}
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
	var s []string
	for _, b := range i.Buckets() {
		s = append(s, string(b))
	}
	return strings.Join(s, " ")
}

// ToCheck returns the directory or file to check.
func (i *internal) ToCheck() string {
	return i.source
}

// OpenDB opens the Bold database.
func (i *internal) OpenDB() {
	if i.db != nil {
		return
	}
	name, err := database.DB()
	if err != nil {
		out.ErrFatal(err)
	}
	if i.db, err = bolt.Open(name, database.FileMode, nil); err != nil {
		out.ErrFatal(err)
	}
}

// Timer returns the time taken since the process timer was instigated.
func (i *internal) Timer() time.Duration {
	return time.Since(i.timer)
}

var (
	ErrNoPath    = errors.New("path does not exist")
	ErrPathExist = errors.New("path exists in the database bucket")
)

// CheckPaths counts the files in the directory to check and the buckets.
func (c *Config) CheckPaths() (ok bool, checkCnt, bucketCnt int) {
	if c.Debug {
		out.Bug("count the files within the paths")
	}
	root := c.ToCheck()
	if c.Debug {
		out.Bug("directory to check: " + root)
	}
	stat, err := os.Stat(root)
	if err != nil {
		return ok, 0, 0
	}
	if !stat.IsDir() {
		return ok, 0, 0
	}
	checkCnt, bucketCnt = 0, 0
	check := func() bool {
		return bucketCnt >= checkCnt/2
	}
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if c.Debug {
			out.Bug("counting: " + path)
		}
		if err != nil {
			return err
		}
		if err := skipDir(d, true); err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		checkCnt++
		return nil
	}); err != nil {
		if c.Debug {
			out.Bug(err.Error())
		}
		return
	}
	if c.Debug {
		s := fmt.Sprintf("all buckets: %s", c.Buckets())
		out.Bug(s)
	}
	for _, b := range c.Buckets() {
		if err := filepath.WalkDir(string(b), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				out.ErrFatal(err)
				return nil
			}
			if c.Debug {
				out.Bug("walking bucket item: " + path)
			}
			if err := skipDir(d, true); err != nil {
				return err
			}
			if !d.Type().IsRegular() {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			bucketCnt++
			if check() {
				return io.EOF
			}
			return nil
		}); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			if c.Debug {
				out.Bug(err.Error())
			}
			return
		}
	}
	return check(), checkCnt, bucketCnt
}

// Print the results of the database comparisons.
func Print(quiet bool, m *database.Matches) string {
	if m == nil {
		return ""
	}
	if len(*m) == 0 {
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

// Clean removes all empty directories from c.Source.
// Directories containing hidden system directories or files are not considered empty.
func (c *Config) Clean() string {
	path := c.ToCheck()
	if path == "" {
		return ""
	}
	var count int
	if err := godirwalk.Walk(path, &godirwalk.Options{
		Unsorted: true,
		Callback: func(_ string, _ *godirwalk.Dirent) error {
			return nil
		},
		PostChildrenCallback: func(osPathname string, _ *godirwalk.Dirent) error {
			s, err := godirwalk.NewScanner(osPathname)
			if err != nil {
				return err
			}
			// Attempt to read only the first directory entry.
			hasAtLeastOneChild := s.Scan()
			// If error reading from directory, wrap up and return.
			if err1 := s.Err(); err1 != nil {
				return err1
			}
			if hasAtLeastOneChild {
				return nil
			}
			if osPathname == path {
				return nil
			}
			count++
			err = os.Remove(osPathname)
			if err == nil {
				count++
			}
			return err
		},
	}); err != nil {
		out.ErrFatal(err)
	}
	if count == 0 {
		return fmt.Sprintln("Nothing required cleaning.")
	}
	return fmt.Sprintf("Removed %d empty directories in: '%s'\n", count, path)
}

// Print the results of a dupe request.
func (c *Config) Print() string {
	if c.Debug {
		out.Bug("print duplicate results")
	}
	w := new(bytes.Buffer)
	for _, path := range c.sources {
		sum, err := read(path)
		if err != nil {
			out.ErrCont(err)
		}
		l := c.lookupOne(sum)
		if l == "" {
			continue
		}
		if l == path {
			continue
		}
		fmt.Fprintln(w, match(path, l))
	}
	return w.String()
}

// Remove all duplicate files from the source directory.
func (c *Config) Remove() string {
	w := new(bytes.Buffer)
	if len(c.sources) == 0 || len(c.compare) == 0 {
		fmt.Fprintln(w, "No duplicate files to remove.")
		return w.String()
	}
	fmt.Fprintln(w)
	for _, path := range c.sources {
		if c.Debug {
			out.Bug("remove read: " + path)
		}
		h, err := read(path)
		if err != nil {
			out.ErrCont(err)
		}
		if l := c.lookupOne(h); l == "" {
			continue
		}
		if c.Debug {
			out.Bug("remove delete: " + path)
		}
		err = os.Remove(path)
		fmt.Fprintln(w, printRM(path, err))
	}
	return w.String()
}

// RemoveAll removes directories from the source directory that do not contain unique MS-DOS or Windows programs.
func (c *Config) RemoveAll(clean bool) string {
	root := c.ToCheck()
	_, err := os.Stat(root)
	if errors.Is(err, os.ErrNotExist) {
		e := fmt.Errorf("%w: %s", ErrNoPath, root)
		out.ErrFatal(e)
	} else if err != nil {
		out.ErrFatal(err)
	}
	if len(c.sources) == 0 {
		return ""
	}

	files, err := os.ReadDir(root)
	if err != nil {
		out.ErrCont(err)
	}

	color.Info.Println("\nRemove ALL files, except for unique Windows/MS-DOS programs ?")
	fmt.Printf("%s %s", color.Secondary.Sprint("target directory:"), root)
	if input := out.YN("Please confirm"); !input {
		os.Exit(0)
	}
	fmt.Println()
	return removeAll(root, files)
}

// Seek sources from the database and print out the matches.
func (c *Config) Seek() string {
	c.init()
	finds, w := []string{}, new(bytes.Buffer)
	for _, path := range c.sources {
		if c.Debug {
			out.Bug("seeking source: " + path)
		}
		h, err := read(path)
		if err != nil {
			out.ErrCont(err)
			return w.String()
		}
		if c.Debug {
			s := fmt.Sprintf("source: %x: %s", h, path)
			out.Bug(s)
		}
		for _, bucket := range c.Buckets() {
			finds, c.files, err = database.Seek(h, string(bucket))
			if err != nil {
				out.ErrCont(err)
				continue
			}
		}
		if len(finds) > 0 {
			for _, find := range finds {
				c.compare[h] = path
				fmt.Fprintln(w, match(path, find))
			}
		}
	}
	return w.String()
}

// Status summarizes the file total and time taken.
func (c *Config) Status() string {
	s := "\n"
	s += color.Secondary.Sprint("Scanned ") +
		color.Primary.Sprintf("%d files", c.files)
	if !c.Test {
		s += color.Secondary.Sprint(", taking ") +
			color.Primary.Sprintf("%s", c.Timer())
	}
	if runtime.GOOS != winOS {
		s += "\n"
	}
	return s
}

// WalkDirs walks the named bucket directories for any new files and saves their checksums to the database.
func (c *Config) WalkDirs() {
	c.init()
	if c.db == nil {
		c.OpenDB()
		defer c.db.Close()
	}
	// walk through the directories provided
	for _, bucket := range c.Buckets() {
		if c.Debug {
			out.Bug("walkdir bucket: " + string(bucket))
		}
		if err := c.WalkDir(bucket); err != nil {
			out.ErrCont(err)
		}
	}
}

// WalkDir walks the named bucket directory for any new files and saves their checksums to the bucket.
func (c *Config) WalkDir(name Bucket) error {
	root := string(name)
	c.init()
	skip := c.skipFiles()
	// open database
	if c.db == nil {
		c.OpenDB()
		defer c.db.Close()
	}
	// create a new bucket if needed
	if err := c.createBucket(name); err != nil {
		return err
	}
	// walk the root directory
	var wg sync.WaitGroup
	if c.Debug {
		out.Bug("walk directory: " + root)
	}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if c.Debug {
			out.Bug("walk file: " + path)
		}
		if err != nil {
			if errors.Is(err, fs.ErrPermission) {
				return nil
			}
			return err
		}
		if err1 := skipDir(d, true); err1 != nil {
			return err1
		}
		if skipFile(d.Name()) {
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if skipSelf(path, skip...) {
			return nil
		}
		c.files++
		if errW := walkDir(root, path, c); errW != nil {
			if errors.Is(errW, ErrPathExist) {
				return nil
			}
			out.ErrFatal(errW)
		}
		printWalk(false, c)
		wg.Add(1)
		go func() {
			c.update(path, root)
			wg.Done()
		}()
		wg.Wait()
		return err
	})
	return err
}

// WalkSource walks the source directory or a file to collect its hashed content for future comparison.
func (c *Config) WalkSource() {
	root := c.ToCheck()
	if c.Debug {
		out.Bug("walksource to check: " + root)
	}
	stat, err := os.Stat(root)
	if errors.Is(err, os.ErrNotExist) {
		e := fmt.Errorf("%w: %s", ErrNoPath, root)
		out.ErrFatal(e)
	} else if err != nil {
		out.ErrFatal(err)
	}
	if !stat.IsDir() {
		c.sources = append(c.sources, root)
		if c.Debug {
			out.Bug("items dupe check: " + strings.Join(c.sources, " "))
		}
		return
	}
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if c.Debug {
			out.Bug(path)
		}
		if err != nil {
			return err
		}
		// skip directories
		if err := skipDir(d, true); err != nil {
			return err
		}
		// skip non-files such as symlinks
		if !d.Type().IsRegular() {
			return nil
		}
		// only append files
		if d.IsDir() {
			return nil
		}
		c.sources = append(c.sources, path)
		return nil
	}); err != nil {
		out.ErrFatal(err)
	}
	if c.Debug {
		out.Bug("directories dupe check: " + strings.Join(c.sources, " "))
	}
}

// createBucket an empty bucket in the database.
func (c *Config) createBucket(name Bucket) error {
	_, err := os.Stat(string(name))
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: %s", ErrNoPath, name)
	} else if err != nil {
		return err
	}
	return c.db.Update(func(tx *bolt.Tx) error {
		if b := tx.Bucket([]byte(name)); b == nil {
			fmt.Printf("New database bucket: '%s'\n", name)
			_, err1 := tx.CreateBucket([]byte(name))
			return err1
		}
		return nil
	})
}

// init initializes the Config maps and database.
func (c *Config) init() {
	// use all the buckets if no specific buckets are provided
	if len(c.Buckets()) == 0 {
		c.SetAllBuckets()
	}
	// normalise bucket names
	for i, b := range c.Buckets() {
		abs, err := filepath.Abs(string(b))
		if err != nil {
			out.ErrCont(err)
			c.Buckets()[i] = ""
			continue
		}
		c.Buckets()[i] = Bucket(abs)
	}
	//
	if c.compare == nil {
		c.compare = make(checksums)
		for i, b := range c.Buckets() {
			c.SetCompares(b)
			if c.Debug {
				s := fmt.Sprintf("init %d: %s", i, b)
				out.Bug(s)
			}
		}
	}
}

// lookup the checksum value in c.compare and return the file path.
func (c *Config) lookupOne(sum checksum) string {
	if c.Debug {
		s := fmt.Sprintf("look up sum in compare (%d): %x", len(c.compare), sum)
		out.Bug(s)
	}
	if f := c.compare[sum]; f != "" {
		return f
	}
	return ""
}

// skipFiles returns c.sources as strings.
func (c *Config) skipFiles() (files []string) {
	files = append(files, c.sources...)
	return files
}

// update gets the checksum of the named file and saves it to the bucket.
func (c *Config) update(name, bucket string) {
	if c.Debug {
		out.Bug("update: " + name)
	}
	// read file, exit if it fails
	sum, err := read(name)
	if err != nil {
		fmt.Println(err)
		return
	}
	if sum == [32]byte{} {
		return
	}
	if err = c.db.Update(func(tx *bolt.Tx) error {
		// directory bucket
		b1 := tx.Bucket([]byte(bucket))
		return b1.Put([]byte(name), sum[:])
	}); err != nil {
		out.ErrCont(err)
	}
	c.compare[sum] = name
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

// containsBin returns true if the root directory contains an MS-DOS or Windows program file.
func containsBin(root string) bool {
	bin := false
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if bin {
			return nil
		}
		if !d.IsDir() {
			if isProgram(d.Name()) {
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

// containsBin returns true if the path to a file contains an MS-DOS or Windows program file extension.
func isProgram(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".com", ".exe":
		return true
	default:
		return false
	}
}

// match prints 'Found duplicate match'.
func match(path, match string) string {
	s := "\n"
	s += color.Info.Sprint("Found duplicate match") +
		":" +
		fmt.Sprintf("\t%s", path) +
		matchItem(match)
	return s
}

// matchItem prints 'Found duplicate match' along with file stat info.
func matchItem(match string) string {
	s := color.Success.Sprint("\n  ⤷ ") +
		fmt.Sprint(match)
	stat, err := os.Stat(match)
	if err != nil {
		return s
	}
	s += "\n    " +
		fmt.Sprintf("%s, ", stat.ModTime().Format(modFmt)) +
		humanize.Bytes(uint64(stat.Size()))
	return s
}

// printRM prints "could not remove:".
func printRM(path string, err error) string {
	if err != nil {
		e := fmt.Errorf("could not remove: %w", err)
		out.ErrCont(e)
		return ""
	}
	return fmt.Sprintf("%s: %s\n", color.Secondary.Sprint("removed"), path)
}

// printWalk prints "Scanning/Looking up".
func printWalk(lookup bool, c *Config) {
	if c.Test || c.Quiet || c.Debug {
		return
	}
	s := "Scanning"
	if lookup {
		s = "Looking up"
	}
	if runtime.GOOS == winOS {
		// color output slows down large scans on Windows
		fmt.Printf("\r%s %d files  ", s, c.files)
	} else {
		fmt.Print("\u001b[2K")
		fmt.Print("\r", color.Secondary.Sprintf("%s ", s),
			color.Primary.Sprintf("%d files ", c.files))
	}
}

// read opens the named file and returns a SHA256 checksum of the data.
func read(name string) (sum checksum, err error) {
	f, err := os.Open(name)
	if err != nil {
		return checksum{}, err
	}
	defer f.Close()

	buf, h := make([]byte, oneMb), sha256.New()
	if _, err := io.CopyBuffer(h, f, buf); err != nil {
		return checksum{}, err
	}
	copy(sum[:], h.Sum(nil))
	return sum, nil
}

// removeAll removes directories that do not contain MS-DOS or Windows programs.
func removeAll(root string, files []fs.DirEntry) string {
	w := new(bytes.Buffer)
	for _, item := range files {
		if !item.IsDir() {
			continue
		}
		path := filepath.Join(root, item.Name())
		if containsBin(path) {
			continue
		}
		err := os.RemoveAll(path)
		fmt.Fprintln(w, printRM(path, err))
	}
	return w.String()
}

// skipDir tells WalkDir to ignore specific system and hidden directories.
func skipDir(d fs.DirEntry, hidden bool) error {
	if !d.IsDir() {
		return nil
	}
	// skip directories
	switch strings.ToLower(d.Name()) {
	// the SkipDir return tells WalkDir to skip all files in these directories
	case ".git", ".cache", ".config", ".local", "node_modules", "__macosx", "appdata":
		return filepath.SkipDir
	default:
		// Unix style hidden directories
		if hidden && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		// Windows system directories
		if hidden && runtime.GOOS == winOS && strings.HasPrefix(d.Name(), "$") {
			return filepath.SkipDir
		}
		return nil
	}
}

// skipFile returns true if the file matches a known Windows or macOS system file.
func skipFile(name string) bool {
	switch strings.ToLower(name) {
	case ".ds_store", ".trashes":
		// macOS
		return true
	case "desktop.ini", "hiberfil.sys", "ntuser.dat", "pagefile.sys", "swapfile.sys", "thumbs.db":
		// Windows
		return true
	}
	// macOS
	return strings.HasPrefix(name, "._")
}

// skipSelf returns true if the path exists in skip.
func skipSelf(path string, skip ...string) bool {
	for _, n := range skip {
		if path == n {
			return true
		}
	}
	return false
}

// walkDir walks the root directory and adds the checksums of the files to c.compare.
func walkDir(root, path string, c *Config) error {
	return c.db.View(func(tx *bolt.Tx) error {
		if !c.Test && !c.Quiet && !c.Debug {
			if runtime.GOOS == winOS {
				// color output slows down large scans on Windows
				fmt.Printf("\rLooking up %d files", c.files)
			} else {
				fmt.Print("\u001b[2K\r", color.Secondary.Sprint("Looking up "),
					color.Primary.Sprintf("%d files", c.files))
			}
		}
		b := tx.Bucket([]byte(root))
		if b == nil {
			return nil
		}
		h := b.Get([]byte(path))
		if len(h) > 0 {
			var sum checksum
			copy(sum[:], h)
			c.compare[sum] = path
			return ErrPathExist
		}
		return nil
	})
}
