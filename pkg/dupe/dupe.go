// © Ben Garrett https://github.com/bengarrett/dupers
package dupe

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
	"time"

	"github.com/bengarrett/dupers/internal/out"
	"github.com/bengarrett/dupers/pkg/database"
	"github.com/bengarrett/dupers/pkg/dupe/internal/archive"
	"github.com/bengarrett/dupers/pkg/dupe/internal/parse"
	"github.com/bodgit/sevenzip"
	"github.com/dustin/go-humanize"
	"github.com/gookit/color"
	"github.com/karrick/godirwalk"
	"github.com/mholt/archiver/v3"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

const (
	WinOS = "windows"

	modFmt = "02 Jan 2006 15:04"
	oneKb  = 1024
	oneMb  = oneKb * oneKb
)

var (
	ErrNoNamedBucket = errors.New("a named bucket is required")
	ErrPathIsFile    = errors.New("path is a file")
	ErrPathExist     = errors.New("path exists in the database bucket")
	ErrPathNoFound   = errors.New("path does not exist")
	ErrNilConfig     = errors.New("config cannot be a nil value")
)

// Config options.
type Config struct {
	Debug bool // Debug spams technobabble to stdout.
	Quiet bool // Quiet the feedback sent to stdout.
	Test  bool // Test toggles the internal unit test mode.
	parse.Scanner
}

// DPrint prints the string to stdout whenever Config.Debug is true.
func (c *Config) DPrint(s string) {
	if !c.Debug {
		return
	}
	fmt.Fprintf(os.Stdout, "∙%s\n", s)
}

// CheckPaths counts the number of files in a directory to check and the number of buckets.
func (c *Config) CheckPaths() (files, buckets int, err error) {
	c.DPrint("count the files within the paths")
	dupeItem := c.GetSource()
	c.DPrint("path to check: " + dupeItem)

	s, err := c.CheckDir(dupeItem)
	if err != nil {
		return 0, 0, err
	}
	if s != dupeItem {
		c.DPrint("path to check was not found: " + dupeItem)
		c.DPrint("will attempt to use: " + s)
		dupeItem = s
	}
	c.DPrint(fmt.Sprintf("all buckets: %s", c.All()))

	buckets = 0
	files, err = c.walkPath(dupeItem)
	if err != nil {
		return 0, 0, err
	}
	for _, b := range c.All() {
		var err error
		buckets, err = c.walkBucket(b, files, buckets)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return 0, 0, err
		}
	}
	if buckets >= files/2 {
		c.DPrint("there seems to be too few files in the buckets")
	}
	return files, buckets, nil
}

func (c *Config) walkBucket(b parse.Bucket, files, buckets int) (int, error) {
	root := string(b)
	return buckets, filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if root == path {
			return nil
		}
		c.DPrint("walking bucket item: " + path)
		if err := skipDir(d); err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		buckets++
		if buckets >= files/2 {
			return io.EOF
		}
		return nil
	})
}

func (c *Config) walkPath(root string) (int, error) {
	checkCnt := 0
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		c.DPrint("counting: " + path)
		if err != nil {
			return err
		}
		if root == path {
			return nil
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
		return 0, err
	}
	return checkCnt, nil
}

// CheckItem stat and returns the named file or directory.
// If it does not exist, it looks up an absolute path and returns the result.
// If the item is a file it returns both the named file and an ErrPathIsFile error.
func (c *Config) CheckDir(name string) (string, error) {
	stat, err := os.Stat(name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			c.DPrint("path is not found, but there is an absolute resolved path")
			s, err1 := filepath.Abs(name)
			if err1 != nil {
				return "", os.ErrNotExist
			}
			return s, nil
		}
		c.DPrint("path is not found")
		return "", err
	}
	if !stat.IsDir() {
		return name, ErrPathIsFile
	}
	return name, nil
}

// Checksum the named file and save it to the bucket.
func (c *Config) Checksum(db *bolt.DB, name, bucket string) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	if len(c.Compare) == 0 {
		c.Compare = make(parse.Checksums)
	}
	c.DPrint("update: " + name)
	// read file, exit if it fails
	sum, err := parse.Read(name)
	if err != nil {
		return err
	}
	if sum == [32]byte{} {
		return nil
	}
	if err = db.Update(func(tx *bolt.Tx) error {
		// directory bucket
		b1 := tx.Bucket([]byte(bucket))
		return b1.Put([]byte(name), sum[:])
	}); err != nil {
		return err
	}
	if c.Compare == nil {
		c.Compare = make(parse.Checksums)
	}
	c.Compare[sum] = name
	return nil
}

// Clean removes all empty directories from c.Source.
// Directories containing hidden system directories or files are not considered empty.
func (c *Config) Clean() error {
	path := c.GetSource()
	if path == "" {
		return nil
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
		return err
	}
	w := os.Stdout
	if count == 0 {
		fmt.Fprintln(w, "Nothing required cleaning.")
		return nil
	}
	fmt.Fprintf(w, "Removed %d empty directories in: '%s'\n", count, path)
	return nil
}

// Print the results of a dupe request.
func (c *Config) Print() (string, error) {
	c.DPrint("print duplicate results")
	c.DPrint(fmt.Sprintf("comparing %d sources against %d unique items to compare",
		len(c.Sources), len(c.Compare)))

	w := new(bytes.Buffer)
	finds := 0
	for _, path := range c.Sources {
		sum, err := parse.Read(path)
		if err != nil {
			return "", err
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
	return w.String(), nil
}

// Remove duplicate files from the source directory.
func (c *Config) Remove() (string, error) {
	w := new(bytes.Buffer)
	if len(c.Sources) == 0 || len(c.Compare) == 0 {
		fmt.Fprintln(w, "No duplicate files to remove.          ")
		return w.String(), nil
	}
	fmt.Fprintln(w)
	for _, path := range c.Sources {
		c.DPrint("remove read: " + path)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			c.DPrint("path is not exist: " + path)
			return "", err
		}
		checksum, err := parse.Read(path)
		if err != nil {
			return "", err
		}
		if l := c.lookupOne(checksum); l == "" {
			continue
		}
		c.DPrint("remove delete: " + path)
		err = os.Remove(path)
		fmt.Fprintln(w, printRM(path, err))
	}
	return w.String(), nil
}

// Removes the directories from the source that do not contain unique MS-DOS or Windows programs.
func (c *Config) Removes(assumeYes bool) (string, error) {
	root := c.GetSource()
	if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("%w: %s", ErrPathNoFound, root)
	} else if err != nil {
		return "", err
	}
	if len(c.Sources) == 0 {
		return "", nil
	}
	files, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}
	if !c.Test {
		w := os.Stdout
		fmt.Fprintf(w, "%s %s\n", color.Secondary.Sprint("Target directory:"), color.Debug.Sprint(root))
		fmt.Fprintln(w, "Delete everything in the target directory, except for directories"+
			"\ncontaining unique Windows or MS-DOS programs and assets?")
		if input := out.AskYN("Please confirm", assumeYes, out.Nil); !input {
			os.Exit(0)
		}
		fmt.Fprintln(w)
	}
	return removes(root, files), nil
}

// Status summarizes the file totals and process duration.
func (c *Config) Status() string {
	p, s, l := message.NewPrinter(language.English), "\r", len(c.Compare)
	if c.Files == 0 && l > 0 {
		// -fast flag
		s += color.Secondary.Sprint("Scanned ") +
			color.Primary.Sprintf("%s unique items", p.Sprint(number.Decimal(l)))
	} else {
		s += color.Secondary.Sprint("Scanned ") +
			color.Primary.Sprintf("%s files", p.Sprint(number.Decimal(c.Files)))
	}
	if !c.Test {
		t := c.Timer().Truncate(time.Millisecond)
		s += color.Secondary.Sprint(", taking ") +
			color.Primary.Sprintf("%v", t)
	}
	if runtime.GOOS != WinOS {
		s += "\n"
	}
	return s
}

// WalkDirs walks the named bucket directories for any new files to add their checksums to the database.
func (c *Config) WalkDirs(db *bolt.DB) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	if err := c.init(db); err != nil {
		return err
	}
	// walk through the directories provided
	for _, bucket := range c.All() {
		s := string(bucket)
		c.DPrint("walkdir bucket: " + s)
		if err := c.WalkDir(db, bucket); err != nil {
			if errors.Is(errors.Unwrap(err), ErrPathNoFound) &&
				errors.Is(database.Exist(db, s), bolt.ErrBucketNotFound) {
				out.StderrCR(err)
				continue
			}
			return err
		}
	}
	// handle any items that exist in the database but not in the file system
	// this would include items added using the `up+` archive scan command
	for _, b := range c.All() {
		if _, err := c.SetCompares(db, b); err != nil {
			return err
		}
	}
	return nil
}

// WalkDir walks the named bucket directory for any new files to add their checksums to the database.
func (c *Config) WalkDir(db *bolt.DB, name parse.Bucket) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	if name == "" {
		return ErrNoNamedBucket
	}
	root := string(name)
	if err := c.init(db); err != nil {
		return err
	}
	skip := c.skipFiles()
	// create a new bucket if needed
	if err := c.create(db, name); err != nil {
		return err
	}
	// walk the root directory
	return c.walkDir(db, root, skip)
}

func (c *Config) walkDir(db *bolt.DB, root string, skip []string) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	c.DPrint("walk directory: " + root)
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if c.Debug {
			s := "walk file"
			if d.IsDir() {
				s = "walk subdirectory"
			}
			c.DPrint(fmt.Sprintf("%s: %s", s, path))
		}
		if err != nil {
			if errors.Is(err, fs.ErrPermission) {
				return nil
			}
			return err
		}
		if path == root {
			return nil
		}
		if err := skipDir(d); err != nil {
			return c.walkDebug(" - skipping directory", err)
		}
		if skipFile(d.Name()) {
			return c.walkDebug(" - skipping file", nil)
		}
		if !d.Type().IsRegular() {
			return c.walkDebug(" - skipping not regular file", nil)
		}
		if skipSelf(path, skip...) {
			return c.walkDebug(" - skipping self item", nil)
		}
		c.Files++
		if err := c.walkCompare(db, root, path); err != nil {
			if errors.Is(err, ErrPathExist) {
				return nil
			}
			return err
		}
		fmt.Fprint(os.Stdout, PrintWalk(false, c))
		if err := c.Checksum(db, path, root); err != nil {
			return err
		}
		return nil
	})
}

func (c *Config) walkDebug(s string, err error) error {
	c.DPrint(s)
	return err
}

// WalkSource walks the source directory or a file to collect the hashed content for a future comparison.
func (c *Config) WalkSource() error {
	root := c.GetSource()
	c.DPrint("walksource to check: " + root)
	stat, err := os.Stat(root)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: %s", ErrPathNoFound, root)
	}
	if err != nil {
		return fmt.Errorf("%w: %s", err, root)
	}
	if !stat.IsDir() {
		c.Sources = append(c.Sources, root)
		c.DPrint("items dupe check: " + strings.Join(c.Sources, " "))
		return nil
	}
	if err := c.walkSource(root); err != nil {
		out.StderrCR(fmt.Errorf("item has a problem: %w", err))
		return nil
	}
	c.DPrint("directories dupe check: " + strings.Join(c.Sources, " "))
	return nil
}

func (c *Config) walkSource(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		c.DPrint(path)
		if err != nil {
			return err
		}
		if root == path {
			return nil
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
		c.Sources = append(c.Sources, path)
		return nil
	})
}

// create a new, empty bucket in the database.
func (c *Config) create(db *bolt.DB, name parse.Bucket) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	if _, err := os.Stat(string(name)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrPathNoFound, name)
		}
		return err
	}
	return db.Update(func(tx *bolt.Tx) error {
		if b := tx.Bucket([]byte(name)); b == nil {
			_, err := tx.CreateBucket([]byte(name))
			return err
		}
		return nil
	})
}

// init initializes the Config maps and database.
func (c *Config) init(db *bolt.DB) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	// use all the buckets if no specific buckets are provided
	if !c.Test && len(c.All()) == 0 {
		if err := c.SetAllBuckets(db); err != nil {
			return err
		}
	}
	// normalise bucket names
	for i, b := range c.All() {
		abs, err := database.Abs(string(b))
		if err != nil {
			out.StderrCR(err)
			c.All()[i] = ""

			continue
		}
		c.All()[i] = parse.Bucket(abs)
	}
	if c.Compare == nil {
		c.Compare = make(parse.Checksums)
	}
	if !c.Test && c.Compare == nil {
		for i, b := range c.All() {
			_, err := c.SetCompares(db, b)
			if err != nil {
				return err
			}
			c.DPrint(fmt.Sprintf("init %d: %s", i, b))
		}
	}
	return nil
}

// lookup the checksum value in c.compare and return the file path.
func (c *Config) lookupOne(sum parse.Checksum) string {
	if len(c.Compare) == 0 {
		c.Compare = make(parse.Checksums)
	}
	c.DPrint(fmt.Sprintf("look up checksum in the compare data, %d items total: %x",
		len(c.Compare), sum))
	if f := c.Compare[sum]; f != "" {
		c.DPrint("lookupOne match: " + f)
		return f
	}
	return ""
}

// skipFiles returns the value of c.sources as strings.
func (c *Config) skipFiles() (files []string) {
	files = append(files, c.Sources...)
	return files
}

// match prints 'Found duplicate match'.
func match(path, match string) string {
	s := "\n"
	s += color.Info.Sprint("Match") +
		":" +
		fmt.Sprintf("\t%s", path) +
		matchItem(match)
	return s
}

// matchItem prints 'Found duplicate match' along with file stat info.
func matchItem(match string) string {
	s := color.Success.Sprint(out.MatchPrefix) +
		fmt.Sprint(match)
	stat, err := os.Stat(match)
	if err != nil {
		return s
	}
	s += "\n    " +
		fmt.Sprintf("\t%s, ", stat.ModTime().Format(modFmt)) +
		humanize.Bytes(uint64(stat.Size()))
	return s
}

// printRM prints "could not remove:".
func printRM(path string, err error) string {
	if err != nil {
		e := fmt.Errorf("could not remove: %w", err)
		out.StderrCR(e)
		return ""
	}
	return fmt.Sprintf("%s: %s", color.Secondary.Sprint("removed"), path)
}

// PrintWalk prints "Scanning/Looking up".
func PrintWalk(lookup bool, c *Config) string {
	if c.Test || c.Quiet || c.Debug {
		return ""
	}
	if lookup {
		return out.Status(c.Files, -1, out.Look)
	}
	return out.Status(c.Files, -1, out.Scan)
}

// removes directories that do not contain MS-DOS or Windows programs.
func removes(root string, files []fs.DirEntry) string {
	w := new(bytes.Buffer)

	for _, item := range files {
		if !item.IsDir() {
			continue
		}
		path := filepath.Join(root, item.Name())
		if exe, err := parse.Executable(path); err != nil {
			out.StderrCR(err)
			continue
		} else if exe {
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
		if runtime.GOOS == WinOS && strings.HasPrefix(d.Name(), "$") {
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
func (c *Config) walkCompare(db *bolt.DB, root, path string) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	if c.Compare == nil {
		c.Compare = make(parse.Checksums)
	}
	return db.View(func(tx *bolt.Tx) error {
		if !c.Test && !c.Quiet && !c.Debug {
			fmt.Fprint(os.Stdout, out.Status(c.Files, -1, out.Scan))
		}
		b := tx.Bucket([]byte(root))
		if b == nil {
			return ErrNoNamedBucket
		}
		h := b.Get([]byte(path))
		c.DPrint(fmt.Sprintf(" - %d/%d items: %x", len(c.Compare), c.Files, h))
		if len(h) > 0 {
			var sum parse.Checksum
			copy(sum[:], h)
			c.Compare[sum] = path
			return ErrPathExist
		}
		return nil
	})
}

// Print the results of the database comparisons.
func Print(quiet, exact bool, term string, m *database.Matches) string {
	return parse.Print(quiet, exact, term, m)
}

// Bucket returns the named string as a Bucket type.
func Bucket(name string) parse.Bucket {
	return parse.Bucket(name)
}

// WalkArchiver walks the bucket directory saving the checksums of new files to the database.
// Any archived files supported by archiver will also have its content hashed.
// Archives within archives are currently left unwalked.
func (c *Config) WalkArchiver(db *bolt.DB, name parse.Bucket) error {
	if name == "" {
		return ErrNoNamedBucket
	}
	root := string(name)

	if err := c.init(db); err != nil {
		return err
	}
	skip := c.skipFiles()
	// create a new bucket if needed
	if err := c.create(db, name); err != nil {
		return err
	}
	// get a list of all the bucket filenames
	if err := c.listItems(db, root); err != nil {
		return err
	}
	// walk the root directory of the bucket
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		c.DPrint("walk file: " + path)
		if err != nil {
			if errors.Is(err, fs.ErrPermission) {
				return nil
			}
			return err
		}
		if root == path {
			return nil
		}
		if err1 := skipDir(d); err1 != nil {
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
		err = c.walkThread(db, parse.Bucket(root), path)
		if err != nil {
			out.StderrCR(err)
		}
		return nil
	})
}

func (c *Config) walkThread(db *bolt.DB, b parse.Bucket, path string) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	// detect archive type by file extension
	mimeExt := strings.ToLower(filepath.Ext(path))
	ok := (archive.MIME(path) != "")
	c.DPrint(fmt.Sprintf("is known extension: %v, %s", ok, mimeExt))
	if !ok {
		// detect archive type by mime type
		mime, err := archive.ReadMIME(path)
		if errors.Is(err, archive.ErrFilename) {
			if mime != "" {
				c.DPrint(fmt.Sprintf("archive not supported: %s: %s", mime, path))
			}
			return nil
		}
		if err != nil {
			return err
		}
		mimeExt = archive.Extension(mime)
	}
	c.Files++
	c.DPrint(fmt.Sprintf("walkCompare #%d", c.Files))
	if errD := c.walkCompare(db, string(b), path); errD != nil {
		if !errors.Is(errD, ErrPathExist) {
			out.ErrFatal(errD)
		}
	}
	// archive reader
	const unknownExt = ""
	switch mimeExt {
	case unknownExt:
		return nil
	case archive.Ext7z:
		return c.Read7Zip(db, b, path)
	default:
		return c.Read(db, b, path, mimeExt)
	}
}

// findItem returns true if the absolute file path is in c.sources.
func (c *Config) findItem(abs string) bool {
	for _, item := range c.Sources {
		if item == abs {
			return true
		}
	}
	return false
}

// listItems sets c.sources to list all the filenames used in the bucket.
func (c *Config) listItems(db *bolt.DB, bucket string) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	c.DPrint("list bucket items: " + bucket)
	abs, err := database.AbsB(bucket)
	if err != nil {
		out.StderrCR(err)
	}
	if err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(abs)
		if b == nil {
			return nil
		}
		err = b.ForEach(func(key, _ []byte) error {
			if bytes.Contains(key, []byte(bucket)) {
				c.Sources = append(c.Sources, string(database.Filepath(key)))
			}
			return nil
		})
		return err
	}); err != nil {
		if errors.Is(err, bolt.ErrBucketNotFound) {
			return fmt.Errorf("%w: '%s'", err, abs)
		}
		return err
	}
	return nil
}

// Read7Zip opens the named 7-Zip archive, hashes and saves the content to the bucket.
func (c *Config) Read7Zip(db *bolt.DB, b parse.Bucket, name string) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	c.DPrint("read 7zip: " + name)
	r, err := sevenzip.OpenReader(name)
	if err != nil {
		return err
	}
	defer r.Close()
	cnt := 0
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		path := filepath.Join(name, f.Name)
		if c.findItem(path) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			out.Stderr(err)
			continue
		}
		defer rc.Close()
		cnt++
		buf, h := make([]byte, oneMb), sha256.New()
		if _, err := io.CopyBuffer(h, rc, buf); err != nil {
			out.Stderr(err)
			continue
		}
		var sum parse.Checksum
		copy(sum[:], h.Sum(nil))
		if err := c.update(db, b, path, sum); err != nil {
			return err
		}
	}
	if cnt > 0 {
		c.DPrint(fmt.Sprintf("read %d items within the 7-Zip archive", cnt))
	}
	return nil
}

// Read opens the named archive, hashes and saves the content to the bucket.
func (c *Config) Read(db *bolt.DB, b parse.Bucket, name, mimeExt string) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	c.DPrint("read archiver: " + name)
	// catch any archiver panics such as opening unsupported ZIP compression formats
	defer c.readRecover(name)
	cnt, lookup := 0, name
	if mimeExt != "" {
		lookup = mimeExt
	}
	// get the format by filename extension
	tars := []string{".gz", ".br", ".bz2", ".lz4", ".sz", ".xz", ".zz", ".zst"}
	for _, t := range tars {
		if strings.HasSuffix(name, ".tar"+t) {
			lookup = ".tar" + t
			break
		}
	}
	f, err := archiver.ByExtension(strings.ToLower(lookup))
	if err != nil {
		out.StderrCR(err)
		return nil
	}
	switch archive.Supported(f) {
	case true:
		w, ok := f.(archiver.Walker)
		if !ok {
			out.StderrCR(fmt.Errorf("%w: %s: %s", archive.ErrType, lookup, name))
			return nil
		}
		cnt, err = c.readWalk(db, b, name, cnt, w)
		if err != nil {
			out.Stderr(err)
		}
	default:
		color.Warn.Printf("Unsupported archive: '%s'\n", name)
		return nil
	}
	if cnt > 0 {
		c.DPrint(fmt.Sprintf("read %d items within the archive", cnt))
	}
	return nil
}

func (c *Config) readWalk(db *bolt.DB, b parse.Bucket, archive string, cnt int, w archiver.Walker) (int, error) {
	if db == nil {
		return -1, bolt.ErrDatabaseNotOpen
	}
	return cnt, w.Walk(archive, func(f archiver.File) error {
		if f.IsDir() {
			return nil
		}
		if !f.FileInfo.Mode().IsRegular() {
			return nil
		}
		if skipFile(f.Name()) {
			return nil
		}
		path := filepath.Join(archive, f.Name())
		if c.findItem(path) {
			return nil
		}
		buf, h := make([]byte, oneMb), sha256.New()
		if _, err := io.CopyBuffer(h, f, buf); err != nil {
			out.Stderr(err)
			return nil
		}
		var sum parse.Checksum
		copy(sum[:], h.Sum(nil))
		if err := c.update(db, b, path, sum); err != nil {
			out.Stderr(err)
		}
		cnt++
		return nil
	})
}

func (c *Config) readRecover(archive string) {
	if err := recover(); err != nil {
		if !c.Quiet {
			if !c.Debug {
				fmt.Fprintln(os.Stdout)
			}
			color.Warn.Printf("Unsupported archive: '%s'\n", archive)
		}
		c.DPrint(fmt.Sprint(err))
	}
}

// update saves the checksum and path values to the bucket.
func (c *Config) update(db *bolt.DB, b parse.Bucket, path string, sum parse.Checksum) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	if len(c.Compare) == 0 {
		c.Compare = make(parse.Checksums)
	}
	c.DPrint("update archiver: " + path)
	if err := db.Update(func(tx *bolt.Tx) error {
		b1 := tx.Bucket([]byte(b))
		if b1 == nil {
			return bolt.ErrBucketNotFound
		}
		return b1.Put([]byte(path), sum[:])
	}); err != nil {
		return err
	}
	c.Compare[sum] = path
	return nil
}
