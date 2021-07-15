// © Ben Garrett https://github.com/bengarrett/dupers

// dupers todo
package dupers

import (
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

	"github.com/bengarrett/dupers/lib/database"
	"github.com/bengarrett/dupers/lib/out"
	"github.com/dustin/go-humanize"
	"github.com/gookit/color"
	bolt "go.etcd.io/bbolt"
)

// Config options for duper.
type Config struct {
	Timer   time.Time
	Buckets []string // buckets to lookup
	Source  string   // directory or file to compare
	Quiet   bool     // reduce the feedback sent to stdout
	db      *bolt.DB // interal Bolt database
	compare hash     // hashes fetched from the database or file system
	files   int      // total files or database items read
	sources []string // files paths to check
	test    bool     // internal unit test mode
}

type hash map[[32]byte]string

const (
	modFmt = "02 Jan 2006 15:04"
	winOS  = "windows"
)

var (
	ErrNoPath    = errors.New("path does not exist")
	ErrPathExist = errors.New("path exists in the database bucket")
)

// Print the results of the database comparisons.
func Print(term string, quiet bool, m *database.Matches) {
	if len(*m) == 0 {
		return
	}
	// collect the bucket names which will be used to sort the results
	buckets, bucket := []string{}, ""
	for _, bucket := range *m {
		if !contains(buckets, bucket) {
			buckets = append(buckets, bucket)
		}
	}
	sort.Strings(buckets)
	for i, buck := range buckets {
		cnt := 0
		if i > 0 {
			fmt.Println()
		}
		// print the matches, the filenames are unsorted
		for file, b := range *m {
			if b != buck {
				continue
			}
			cnt++
			if b != bucket {
				bucket = b
				if !quiet {
					if cnt > 1 {
						fmt.Println()
					}
					fmt.Printf("%s: %s\n", color.Info.Sprint("Results from"), b)
				}
			}
			if quiet {
				fmt.Printf("%s\n", file)
				continue
			}
			if cnt == 1 {
				fmt.Printf("%s%s\n", color.Success.Sprint("  ⤷\t"), file)
				continue
			}
			fmt.Printf("  %s%s\t%s\n", color.Primary.Sprint(cnt), color.Secondary.Sprint("."), file)
		}
	}
}

func contains(s []string, find string) bool {
	for _, item := range s {
		if find == item {
			return true
		}
	}
	return false
}

// init initializes the Config maps and database.
func (c *Config) init() {
	// use all the buckets if no specific buckets are provided
	if len(c.Buckets) == 0 {
		var err error
		c.Buckets, err = database.Buckets()
		if err != nil {
			out.ErrFatal(err)
		}
	}
	// normalise bucket names
	for i, b := range c.Buckets {
		abs, err := filepath.Abs(b)
		if err != nil {
			out.ErrCont(err)
			c.Buckets[i] = ""
			continue
		}
		c.Buckets[i] = abs
	}
	if c.compare == nil {
		c.compare = make(hash)
	}
}

// PurgeSrc removes all files and directories from the source directory that are not unique MS-DOS or Windows programs.
func (c *Config) PurgeSrc() {
	root, err := filepath.Abs(c.Source)
	if err != nil {
		out.ErrFatal(err)
	}

	_, err = os.Stat(root)
	if errors.Is(err, os.ErrNotExist) {
		e := fmt.Errorf("%w: %s", ErrNoPath, root)
		out.ErrFatal(e)
	} else if err != nil {
		out.ErrFatal(err)
	}

	if len(c.sources) == 0 {
		return
	}

	files, err := os.ReadDir(root)
	if err != nil {
		out.ErrCont(err)
	}
	for _, item := range files {
		if !item.IsDir() {
			continue
		}
		path := filepath.Join(root, item.Name())
		if containsBin(path) {
			continue
		}
		err := os.RemoveAll(path)
		printRM(path, err)
	}
}

func printRM(path string, err error) {
	if err != nil {
		e := fmt.Errorf("could not remove: %w", err)
		out.ErrCont(e)
		return
	}
	fmt.Printf("%s: %s\n", color.Secondary.Sprint("removed"), path)
}

func containsBin(root string) bool {
	bin := false
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
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

func isProgram(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".com", ".exe":
		return true
	default:
		return false
	}
}

// Print the results of a dupe request.
func (c *Config) Print() {
	for _, path := range c.sources {
		h, err := read(path)
		if err != nil {
			out.ErrCont(err)
		}
		l := c.lookupOne(h)
		if l == "" {
			continue
		}
		if l == path {
			continue
		}
		fmt.Println(match(path, l))
	}
}

func (c *Config) lookupOne(h [32]byte) string {
	if f := c.compare[h]; f != "" {
		return f
	}
	return ""
}

func match(path, match string) string {
	s := "\n"
	s += color.Info.Sprint("Found duplicate match") +
		":" +
		fmt.Sprintf("\t%s", path) +
		matchItem(match)
	return s
}

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

// Remove all duplicate files from the source directory.
func (c *Config) Remove() {
	if len(c.sources) == 0 || len(c.compare) == 0 {
		fmt.Println("No duplicate files to remove.")
		return
	}
	fmt.Println()
	for _, path := range c.sources {
		h, err := read(path)
		if err != nil {
			out.ErrCont(err)
		}
		l := c.lookupOne(h)
		if l == "" {
			continue
		}
		err = os.Remove(path)
		printRM(path, err)
	}
}

// Seek sources from the database and print out the matches.
func (c *Config) Seek() {
	c.init()
	var finds []string
	for _, path := range c.sources {
		h, err := read(path)
		if err != nil {
			out.ErrCont(err)
			return
		}
		for _, bucket := range c.Buckets {
			finds, c.files, err = database.Seek(h, bucket)
			if err != nil {
				out.ErrCont(err)
				continue
			}
		}
		if len(finds) > 0 {
			for _, find := range finds {
				c.compare[h] = path
				fmt.Println(match(path, find))
			}
		}
	}
}

// Status summarizes the file total and time taken.
func (c *Config) Status() string {
	s := "\n"
	s += color.Secondary.Sprint("Scanned ") +
		color.Primary.Sprintf("%d files", c.files)
	if !c.test {
		s += color.Secondary.Sprint(", taking ") +
			color.Primary.Sprintf("%s", time.Since(c.Timer))
	}
	if runtime.GOOS != winOS {
		s += "\n"
	}
	return s
}

// WalkDirs walks the directories provided by the arguments for zip archives to extract any found comments.
func (c *Config) WalkDirs() {
	c.init()
	// open database
	if c.db == nil {
		name, err := database.DB()
		if err != nil {
			out.ErrFatal(err)
		}
		if c.db, err = bolt.Open(name, database.FileMode, nil); err != nil {
			out.ErrFatal(err)
		}
		defer c.db.Close()
	}
	// walk through the directories provided
	for _, bucket := range c.Buckets {
		if err := c.WalkDir(bucket); err != nil {
			out.ErrCont(err)
		}
	}
}

// WalkDir walks the root directory for zip archives and to extract any found comments.
func (c *Config) WalkDir(root string) error {
	c.init()
	skip := c.skipFiles()
	// open database
	if c.db == nil {
		name, err := database.DB()
		if err != nil {
			out.ErrFatal(err)
		}
		if c.db, err = bolt.Open(name, database.FileMode, nil); err != nil {
			out.ErrFatal(err)
		}
		defer c.db.Close()
	}
	// create a new bucket if needed
	if err := c.createBucket(root); err != nil {
		return err
	}
	// walk the root directory
	var wg sync.WaitGroup
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrPermission) {
				return nil
			}
			return err
		}
		// skip directories
		if err1 := skipDir(d, true); err1 != nil {
			return err1
		}
		// skip files
		if skipFile(d) {
			return nil
		}
		// skip non-files such as symlinks
		if !d.Type().IsRegular() {
			return nil
		}
		// skip self file matches
		if skipSelf(path, skip) {
			return nil
		}
		// walk the directories
		c.files++
		if errD := walkDir(root, path, c); errD != nil {
			if errors.Is(errD, ErrPathExist) {
				return nil
			}
			out.ErrFatal(errD)
		}
		printWalk(false, c)
		// hash the file
		wg.Add(1)
		go c.update(path, root, &wg)
		wg.Wait()
		return err
	})
	return err
}

func printWalk(lookup bool, c *Config) {
	if c.test || c.Quiet {
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

func skipFile(d fs.DirEntry) bool {
	switch strings.ToLower(d.Name()) {
	case ".ds_store", ".trashes":
		// macOS
		return true
	case "desktop.ini", "hiberfil.sys", "ntuser.dat", "pagefile.sys", "swapfile.sys", "thumbs.db":
		// Windows
		return true
	}
	return false
}

func skipSelf(path string, skip []string) bool {
	for _, n := range skip {
		if path == n {
			return true
		}
	}
	return false
}

func walkDir(root, path string, c *Config) error {
	return c.db.View(func(tx *bolt.Tx) error {
		if !c.test && !c.Quiet {
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
			var hash [32]byte
			copy(hash[:], h)
			c.compare[hash] = path
			return ErrPathExist
		}
		return nil
	})
}

func (c *Config) createBucket(root string) error {
	_, err := os.Stat(root)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: %s", ErrNoPath, root)
	} else if err != nil {
		return err
	}
	return c.db.Update(func(tx *bolt.Tx) error {
		if b := tx.Bucket([]byte(root)); b == nil {
			fmt.Printf("New database bucket: '%s'\n", root)
			_, err1 := tx.CreateBucket([]byte(root))
			return err1
		}
		return nil
	})
}

// WalkSource walks the source directory or a file to collect its hashed content for future comparison.
func (c *Config) WalkSource() {
	root, err := filepath.Abs(c.Source)
	if err != nil {
		out.ErrFatal(err)
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
		return
	}

	if err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
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
}

func (c *Config) update(path, root string, wg *sync.WaitGroup) {
	defer wg.Done()
	h, err := read(path)
	if err != nil {
		fmt.Println(err)
		return
	}
	if h == [32]byte{} {
		return
	}

	if err = c.db.Update(func(tx *bolt.Tx) error {
		// directory bucket
		b1 := tx.Bucket([]byte(root))
		return b1.Put([]byte(path), h[:])
	}); err != nil {
		out.ErrCont(err)
	}
	c.compare[h] = path
}

func (c *Config) skipFiles() (files []string) {
	files = append(files, c.sources...)
	return files
}

func read(path string) (hash [32]byte, err error) {
	const oneKb = 1024

	f, err := os.Open(path)
	if err != nil {
		return hash, err
	}
	defer f.Close()

	buf, h := make([]byte, oneKb*oneKb), sha256.New()
	if _, err := io.CopyBuffer(h, f, buf); err != nil {
		return hash, err
	}
	copy(hash[:], h.Sum(nil))
	return hash, nil
}
