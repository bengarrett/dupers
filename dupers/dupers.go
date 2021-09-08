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
	"strings"
	"sync"

	"github.com/bengarrett/dupers/database"
	"github.com/bengarrett/dupers/out"
	"github.com/dustin/go-humanize"
	"github.com/gookit/color"
	"github.com/karrick/godirwalk"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
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

var (
	ErrNoBucket    = errors.New("a named bucket is required")
	ErrPathExist   = errors.New("path exists in the database bucket")
	ErrPathNoFound = errors.New("path does not exist")
)

// CheckPaths counts the files in the directory to check and the buckets.
func (c *Config) CheckPaths() (ok bool, checkCnt, bucketCnt int) { //nolint: gocyclo
	const notDirectory = true
	if c.Debug {
		out.Bug("count the files within the paths")
	}
	root := c.ToCheck()
	if c.Debug {
		out.Bug("path to check: " + root)
	}
	stat, err := os.Stat(root)
	if err != nil {
		if c.Debug {
			out.Bug("path is not found")
		}
		return notDirectory, 0, 0
	}
	if !stat.IsDir() {
		if c.Debug {
			out.Bug("path is a file")
		}
		return notDirectory, 0, 0
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
		if err := skipDir(d); err != nil {
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
			if err := skipDir(d); err != nil {
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
		}
	}
	return check(), checkCnt, bucketCnt
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
		s := fmt.Sprintf("comparing %d sources against %d unquie items to compare",
			len(c.sources), len(c.compare))
		out.Bug(s)
	}
	w := new(bytes.Buffer)
	finds := 0
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
		finds++
		fmt.Fprintln(w, match(path, l))
	}
	if finds == 0 {
		fmt.Fprintln(w, color.Info.Sprint("\rNo duplicate files found.          "))
	}
	return w.String()
}

// Remove all duplicate files from the source directory.
func (c *Config) Remove() string {
	w := new(bytes.Buffer)
	if len(c.sources) == 0 || len(c.compare) == 0 {
		fmt.Fprintln(w, "No duplicate files to remove.          ")
		return w.String()
	}
	fmt.Fprintln(w)
	for _, path := range c.sources {
		if c.Debug {
			out.Bug("remove read: " + path)
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if c.Debug {
				out.Bug("path is not exist: " + path)
			}
			continue
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
func (c *Config) RemoveAll() string {
	root := c.ToCheck()
	_, err := os.Stat(root)
	if errors.Is(err, os.ErrNotExist) {
		e := fmt.Errorf("%w: %s", ErrPathNoFound, root)
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
	if !c.Test {
		fmt.Printf("%s %s\n", color.Secondary.Sprint("Target directory:"), color.Debug.Sprint(root))
		fmt.Println("Delete everything in the target directory, except for directories\ncontaining unique Windows or MS-DOS programs and assets?")
		if input := out.YN("Please confirm", out.Nil); !input {
			os.Exit(0)
		}
		fmt.Println()
	}
	return removeAll(root, files)
}

// Status summarizes the file total and time taken.
func (c *Config) Status() string {
	p, s, l := message.NewPrinter(language.English), "\r", len(c.compare)
	if c.files == 0 && l > 0 {
		// -fast flag
		s += color.Secondary.Sprint("Scanned ") +
			color.Primary.Sprintf("%s unique items", p.Sprint(number.Decimal(l)))
	} else {
		s += color.Secondary.Sprint("Scanned ") +
			color.Primary.Sprintf("%s files", p.Sprint(number.Decimal(c.files)))
	}
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
	if !c.Test && c.db == nil {
		c.OpenWrite()
		defer c.db.Close()
	}
	// walk through the directories provided
	redo := []Bucket{}
	for _, bucket := range c.Buckets() {
		s := string(bucket)
		if c.Debug {
			out.Bug("walkdir bucket: " + s)
		}
		if err := c.WalkDir(bucket); err != nil {
			if errors.Is(errors.Unwrap(err), ErrPathNoFound) &&
				errors.Is(database.Exist(s, c.db), database.ErrBucketNotFound) {
				out.ErrCont(err)
				continue
			}
			redo = append(redo, bucket)
			out.ErrCont(err)
		}
	}
	// handle buckets that don't exist on the file system
	for _, bucket := range redo {
		if c.Debug {
			out.Bug("redoing bucket: " + string(bucket))
		}
		c.SetCompares(bucket)
	}
}

// WalkDir walks the named bucket directory for any new files and saves their checksums to the bucket.
func (c *Config) WalkDir(name Bucket) error { //nolint:gocyclo
	if name == "" {
		return ErrNoBucket
	}
	root := string(name)
	c.init()
	skip := c.skipFiles()
	// open database
	if !c.Test && c.db == nil {
		c.OpenWrite()
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
			s := "walk file"
			if d.IsDir() {
				s = "walk subdirectory"
			}
			out.Bug(fmt.Sprintf("%s: %s", s, path))
		}
		if err != nil {
			if errors.Is(err, fs.ErrPermission) {
				return nil
			}
			return err
		}
		if err1 := skipDir(d); err1 != nil {
			if c.Debug {
				out.Bug(" - skipping directory")
			}
			return err1
		}
		if skipFile(d.Name()) {
			if c.Debug {
				out.Bug(" - skipping file")
			}
			return nil
		}
		if !d.Type().IsRegular() {
			if c.Debug {
				out.Bug(" - skipping not regular file")
			}
			return nil
		}
		if skipSelf(path, skip...) {
			if c.Debug {
				out.Bug(" - skipping self item")
			}
			return nil
		}
		c.files++
		if errW := walkCompare(root, path, c); errW != nil {
			if errors.Is(errW, ErrPathExist) {
				return nil
			}
			out.ErrFatal(errW)
		}
		fmt.Print(printWalk(false, c))
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
func (c *Config) WalkSource() error {
	root := c.ToCheck()
	if c.Debug {
		out.Bug("walksource to check: " + root)
	}
	stat, err := os.Stat(root)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: %s", ErrPathNoFound, root)
	} else if err != nil {
		return fmt.Errorf("%w: %s", err, root)
	}
	if !stat.IsDir() {
		c.sources = append(c.sources, root)
		if c.Debug {
			out.Bug("items dupe check: " + strings.Join(c.sources, " "))
		}
		return nil
	}
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if c.Debug {
			out.Bug(path)
		}
		if err != nil {
			return err
		}
		// skip directories
		if err := skipDir(d); err != nil {
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
		out.ErrCont(fmt.Errorf("item has a problem: %w", err))
		return nil
	}
	if c.Debug {
		out.Bug("directories dupe check: " + strings.Join(c.sources, " "))
	}
	return nil
}

// createBucket an empty bucket in the database.
func (c *Config) createBucket(name Bucket) error {
	_, err := os.Stat(string(name))
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: %s", ErrPathNoFound, name)
	} else if err != nil {
		return err
	}
	if c.db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	return c.db.Update(func(tx *bolt.Tx) error {
		if b := tx.Bucket([]byte(name)); b == nil {
			_, err1 := tx.CreateBucket([]byte(name))
			return err1
		}
		return nil
	})
}

// init initializes the Config maps and database.
func (c *Config) init() {
	// use all the buckets if no specific buckets are provided
	if !c.Test && len(c.Buckets()) == 0 {
		c.SetAllBuckets()
	}
	// normalise bucket names
	for i, b := range c.Buckets() {
		abs, err := database.Abs(string(b))
		if err != nil {
			out.ErrCont(err)
			c.Buckets()[i] = ""
			continue
		}
		c.Buckets()[i] = Bucket(abs)
	}
	if c.compare == nil {
		c.compare = make(checksums)
	}
	if !c.Test && c.compare == nil {
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
		s := fmt.Sprintf("look up checksum in the compare data, %d items total: %x", len(c.compare), sum)
		out.Bug(s)
	}
	if f := c.compare[sum]; f != "" {
		if c.Debug {
			out.Bug("lookupOne match: " + f)
		}
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
	if c.db == nil {
		out.ErrCont(bolt.ErrDatabaseNotOpen)
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
	return fmt.Sprintf("%s: %s", color.Secondary.Sprint("removed"), path)
}

// printWalk prints "Scanning/Looking up".
func printWalk(lookup bool, c *Config) string {
	if c.Test || c.Quiet || c.Debug {
		return ""
	}
	if lookup {
		return out.Status(c.files, -1, out.Look)
	}
	return out.Status(c.files, -1, out.Scan)
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
func skipDir(d fs.DirEntry) error {
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
		if strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		// Windows system directories
		if runtime.GOOS == winOS && strings.HasPrefix(d.Name(), "$") {
			return filepath.SkipDir
		}
		return nil
	}
}

// skipFile returns true if the file matches a known Windows or macOS system file.
func skipFile(name string) bool {
	const macOS, windows, macOSExtension = true, true, "._"
	switch strings.ToLower(name) {
	case ".ds_store", ".trashes":
		return macOS
	case "desktop.ini", "hiberfil.sys", "ntuser.dat", "pagefile.sys", "swapfile.sys", "thumbs.db":
		return windows
	}
	return strings.HasPrefix(name, macOSExtension)
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

// walkCompare walks the root directory and adds the checksums of the files to c.compare.
func walkCompare(root, path string, c *Config) error {
	if c.db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	if c.compare == nil {
		c.compare = make(checksums)
	}
	return c.db.View(func(tx *bolt.Tx) error {
		if !c.Test && !c.Quiet && !c.Debug {
			fmt.Print(out.Status(c.files, -1, out.Scan))
		}
		b := tx.Bucket([]byte(root))
		if b == nil {
			return ErrNoBucket
		}
		h := b.Get([]byte(path))
		if c.Debug {
			out.Bug(fmt.Sprintf(" - %d/%d items: %x", len(c.compare), c.files, h))
		}
		if len(h) > 0 {
			var sum checksum
			copy(sum[:], h)
			c.compare[sum] = path
			return ErrPathExist
		}
		return nil
	})
}
