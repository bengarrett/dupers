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
	"slices"
	"strings"
	"time"

	"github.com/bengarrett/dupers/internal/printer"
	"github.com/bengarrett/dupers/pkg/database"
	"github.com/bengarrett/dupers/pkg/dupe/internal/archive"
	"github.com/bengarrett/dupers/pkg/dupe/parse"
	"github.com/bodgit/sevenzip"
	"github.com/dustin/go-humanize"
	"github.com/gookit/color"
	"github.com/karrick/godirwalk"
	"github.com/mholt/archiver/v3"
	bolt "go.etcd.io/bbolt"
	bberr "go.etcd.io/bbolt/errors"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

const (
	WinOS = "windows"

	errRead = "error reading: "
	modFmt  = "02 Jan 2006 15:04"
	oneKb   = 1024
	oneMb   = oneKb * oneKb
)

var (
	ErrFileEmpty     = errors.New("file is empty, being 0 byte in size")
	ErrNilConfig     = errors.New("config cannot be nil")
	ErrNoMatch       = errors.New("no match found")
	ErrNoNamedBucket = errors.New("a named bucket is required")
	ErrPathEmpty     = errors.New("path is empty")
	ErrPathIsFile    = errors.New("path is a file")
	ErrPathExist     = errors.New("path exists in the database bucket")
	ErrPathNoFound   = errors.New("path does not exist")
)

func ignore(err error) {
	_, _ = fmt.Fprint(io.Discard, err)
}

func printl(w io.Writer, a ...any) {
	_, _ = fmt.Fprintln(w, a...)
}

func printf(w io.Writer, format string, a ...any) {
	_, _ = fmt.Fprintf(w, format, a...)
}

// Config options.
type Config struct {
	parse.Scanner

	Debug bool // Debug spams technobabble to stdout.
	Quiet bool // Quiet the feedback sent to stdout.
	Yes   bool // Yes is assumed for all user questions and prompts.
	Test  bool // Test toggles the internal unit test mode.
}

// Debugger prints the string to stdout whenever Config.Debug is true.
func (c *Config) Debugger(s string) {
	c.Writer(os.Stdout, s)
}

// Writer writes the string to the io.writer whenever Config.Debug is true.
func (c *Config) Writer(w io.Writer, s string) {
	if !c.Debug {
		return
	}
	printf(w, "∙%s\n", s)
}

// StatSource returns the number of files in the source directory to check.
// If the source is a file, files will always equal 1.
//
// It returns the following values in order,
//   - boolean is directory value
//   - int is the number of files
//   - int verse is the nunber of items in the buckets for the dupe check
func (c *Config) StatSource() (bool, int, int, error) {
	c.Debugger("count the files within the paths")
	src := c.GetSource()
	c.Debugger("path to check: " + src)
	versus := 0
	isDir, files, err := c.statSource()
	if err != nil {
		return isDir, 0, 0, err
	}
	c.Debugger(fmt.Sprintf("all buckets: %s", c.Buckets))
	for _, bucket := range c.Buckets {
		var err error
		versus, err = c.walkBucket(bucket, versus)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return isDir, 0, 0, err
		}
	}
	if isDir && versus >= files/2 {
		c.Debugger("there seems to be too few files in the buckets")
	}
	return isDir, files, versus, nil
}

// Check stats and returns the named file or directory.
// If it does not exist, it looks up an absolute path and returns the result.
// If the item is a file it returns both the named file and an ErrPathIsFile error.
func (c *Config) Check(name string) (bool, string, error) {
	if name == "" {
		return false, "", ErrPathEmpty
	}
	stat, err := os.Stat(name)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			c.Debugger("path is not found")
			return false, "", err
		}
		c.Debugger("path is not found, but there is an absolute resolved path")
		name, err = filepath.Abs(name)
		if err != nil {
			return false, "", os.ErrNotExist
		}
		stat, err = os.Stat(name)
		if err != nil {
			return false, "", err
		}
	}
	if !stat.IsDir() {
		if stat.Size() == 0 {
			return false, name, ErrFileEmpty
		}
		return false, name, nil
	}
	return true, name, nil
}

// Checksum the named file and save it to the bucket.
func (c *Config) Checksum(db *bolt.DB, name, bucket string) error {
	if db == nil {
		return bberr.ErrDatabaseNotOpen
	}
	if len(c.Compare) == 0 {
		c.Compare = make(parse.Checksums)
	}
	c.Debugger("update: " + name)
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
func (c *Config) Clean(w io.Writer) error {
	c.Debugger("remove all empty directories.")
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
	if count == 0 {
		printl(w, "No empty directories required removal.")
		return nil
	}
	printf(w, "Removed %d empty directories in: '%s'\n", count, path)
	return nil
}

// Print the results of a dupe request.
func (c *Config) Print() (string, error) { //nolint:gocognit
	c.Debugger(fmt.Sprintf("print duplicate results\ncomparing %d sources against %d unique items to compare",
		len(c.Sources), len(c.Compare)))

	w := new(bytes.Buffer)
	finds := 0
	for _, root := range c.Sources {
		info, err := os.Stat(root)
		if err != nil {
			return "", err
		}
		if !info.IsDir() {
			if err := c.printer(w, root); err != nil {
				if errors.Is(err, ErrNoMatch) {
					continue
				}
				return "", err
			}
			finds++
			continue
		}
		c.Debugger("parse read path: " + root)
		if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			c.Debugger("walker: " + path)
			if err != nil {
				c.Debugger("stat sources error: " + err.Error())
				return err
			}
			if root == path {
				return nil
			}
			if err := SkipFS(true, false, true, d); err != nil {
				ignore(err)
				return nil
			}
			if err := c.printer(w, root); err != nil {
				if errors.Is(err, ErrNoMatch) {
					return nil
				}
				return err
			}
			finds++
			return nil
		}); err != nil {
			continue
		}
	}
	if finds == 0 {
		printl(w, color.Info.Sprint("\rNo duplicate files found.          "))
	}
	return w.String(), nil
}

// Remove duplicate files from the source directory.
func (c *Config) Remove() (string, error) {
	c.Debugger("remove all duplicate files.")
	w := new(bytes.Buffer)
	if len(c.Sources) == 0 || len(c.Compare) == 0 {
		printl(w, "No duplicate files to remove.          ")
		return w.String(), nil
	}
	printl(w)
	for i, path := range c.Sources {
		c.Debugger(fmt.Sprintf(" %d. remove read: %s", i, path))
		stat, err := os.Stat(path)
		if os.IsNotExist(err) {
			c.Debugger("path is not exist: " + path)
			return "", err
		}
		if stat == nil || stat.IsDir() {
			continue
		}
		checksum, err := parse.Read(path)
		if err != nil {
			return "", err
		}
		if l := c.lookupOne(checksum); l == "" {
			continue
		}
		err = os.Remove(path)
		printl(w, PrintRM(path, err))
	}
	return w.String(), nil
}

// Removes the directories from the source that do not contain unique MS-DOS or Windows programs.
// The strings contains the path of any non-deletable files.
func (c *Config) Removes() ([]string, error) {
	c.Debugger("removes directories that don't contain any DOS or Windows apps.")
	root := c.GetSource()
	if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%w: %s", ErrPathNoFound, root)
	} else if err != nil {
		return nil, err
	}
	if len(c.Sources) == 0 {
		return nil, nil
	}
	files, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	w := os.Stdout
	if !c.Test {
		printf(w, "%s %s\n", color.Secondary.Sprint("Target directory:"), color.Debug.Sprint(root))
		printl(w, "Delete everything in the target directory, except for directories"+
			"\ncontaining unique Windows or MS-DOS programs and assets?")
		if input := printer.AskYN("Please confirm", c.Yes, printer.Nil); !input {
			os.Exit(0)
		}
		printl(w)
	}
	return Removes(w, root, files)
}

// Removes directories that do not contain MS-DOS or Windows programs.
// The strings contains the path of any undeletable files.
func Removes(w io.Writer, root string, files []fs.DirEntry) ([]string, error) {
	s := []string{}
	for _, item := range files {
		path := filepath.Join(root, item.Name())
		if !item.IsDir() {
			err := os.Remove(path)
			printl(w, PrintRM(path, err))
			continue
		}
		exe, err := parse.Executable(path)
		if err != nil {
			printer.StderrCR(err)
			continue
		}
		if exe {
			continue
		}
		err = os.RemoveAll(path)
		printl(w, PrintRM(fmt.Sprintf("%s%s",
			path, string(filepath.Separator)), err))
		if err != nil {
			s = append(s, path)
			continue
		}
	}
	return s, nil
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
		return bberr.ErrDatabaseNotOpen
	}
	if err := c.init(db); err != nil {
		return err
	}
	// walk through the directories provided
	for _, bucket := range c.Buckets {
		s := string(bucket)
		c.Debugger("walkdir bucket: " + s)
		if err := c.WalkDir(db, bucket); err != nil {
			if errors.Is(errors.Unwrap(err), ErrPathNoFound) &&
				errors.Is(database.Exist(db, s), bberr.ErrBucketNotFound) {
				printer.StderrCR(err)
				continue
			}
			return err
		}
	}
	// handle any items that exist in the database but not in the file system
	// this would include items added using the `up+` archive scan command
	for _, b := range c.Buckets {
		if _, err := c.SetCompares(db, b); err != nil {
			return err
		}
	}
	return nil
}

// WalkDir walks the named bucket directory for any new files to add their checksums to the database.
func (c *Config) WalkDir(db *bolt.DB, name parse.Bucket) error {
	if db == nil {
		return bberr.ErrDatabaseNotOpen
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

// WalkSource walks the source directory or a file to collect the hashed content for a future comparison.
func (c *Config) WalkSource() error {
	isDir, _, err := c.statSource()
	if err != nil {
		return err
	}
	if !isDir {
		c.Debugger("items dupe check: " + strings.Join(c.Sources, " "))
		return nil
	}
	root := c.GetSource()
	if root == "" {
		return parse.ErrNoSource
	}
	c.Debugger("walksource to check: " + root)
	if err := c.statSources(root); err != nil {
		printer.StderrCR(fmt.Errorf("item has a problem: %w", err))
		return nil
	}
	c.Debugger("directories dupe check: " + strings.Join(c.Sources, " "))
	return nil
}

// Match prints 'Found duplicate match'.
func Match(path, match string) string {
	if match == "" {
		return ""
	}
	s := "\n"
	s += color.Info.Sprint("Match") +
		":" + "\t" + path + matchItem(match)
	return s
}

// MatchItem prints 'Found duplicate match' along with file stat info.
func matchItem(match string) string {
	matches := color.Success.Sprint(printer.MatchPrefix) +
		match
	if match == "" {
		return ""
	}
	stat, err := os.Stat(match)
	if err != nil {
		return matches
	}
	matches += "\n    " +
		fmt.Sprintf("\t%s, ", stat.ModTime().Format(modFmt)) +
		humanize.Bytes(safesize(stat.Size()))
	return matches
}

func safesize(i int64) uint64 {
	if i < 0 {
		return 0
	}
	return uint64(i)
}

// PrintRM prints "could not remove:".
func PrintRM(path string, err error) string {
	if err != nil {
		e := fmt.Errorf("could not remove: %w", err)
		printer.StderrCR(e)
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
		return printer.Status(c.Files, -1, printer.Look)
	}
	return printer.Status(c.Files, -1, printer.Scan)
}

// SkipFS tells WalkDir to ignore specific files and directories.
// A true dir value will skip directories.
// A true file value will skip OS specific system files.
// A true regular value will skip all non-files such as symlinks.
func SkipFS(dir, file, regular bool, d fs.DirEntry) error {
	if dir && d.IsDir() {
		// skip directories
		return filepath.SkipDir
	}
	if file && SkipFile(d.Name()) {
		// skip specific system files
		return filepath.SkipDir
	}
	if regular && !d.Type().IsRegular() {
		// skip all non-files such as symlinks
		return filepath.SkipDir
	}
	// note: this skips both directories and filenames sharing specific system dirs.
	return SkipDirs(d.Name())
}

// SkipDirs tells WalkDir to ignore specific system and hidden directories.
func SkipDirs(name string) error {
	// skip directories
	switch strings.ToLower(name) {
	// the SkipDir return tells WalkDir to skip all files in these directories
	case ".git", ".cache", ".config", ".local", "node_modules", "__macosx", "appdata":
		return filepath.SkipDir
	default:
		// Unix style hidden directories
		if strings.HasPrefix(name, ".") {
			return filepath.SkipDir
		}
		// Windows system directories
		if runtime.GOOS == WinOS && strings.HasPrefix(name, "$") {
			return filepath.SkipDir
		}
		return nil
	}
}

// SkipFile returns true if the file matches a known Windows or macOS system file.
func SkipFile(name string) bool {
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
	return slices.Contains(skip, path)
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
		c.Debugger("walk file: " + path)
		if err != nil {
			if errors.Is(err, fs.ErrPermission) {
				return nil
			}
			c.Debugger(errRead + err.Error())
			return err
		}
		if root == path || skipSelf(path, skip...) {
			return nil
		}
		if err := SkipFS(false, true, true, d); err != nil {
			ignore(err)
			return nil
		}
		err = c.readArchive(db, parse.Bucket(root), path)
		if err != nil {
			printer.StderrCR(err)
		}
		return nil
	})
}

// Read7Zip opens the named 7-Zip archive, hashes and saves the content to the bucket.
func (c *Config) Read7Zip(db *bolt.DB, b parse.Bucket, name string) error {
	if db == nil {
		return bberr.ErrDatabaseNotOpen
	}
	c.Debugger("read 7zip: " + name)
	r, err := sevenzip.OpenReader(name)
	if err != nil {
		return err
	}
	defer func() {
		_ = r.Close()
	}()
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
			printer.Stderr(err)
			continue
		}
		defer func() {
			_ = rc.Close()
		}()
		cnt++
		buf, h := make([]byte, oneMb), sha256.New()
		if _, err := io.CopyBuffer(h, rc, buf); err != nil {
			printer.Stderr(err)
			continue
		}
		var sum parse.Checksum
		copy(sum[:], h.Sum(nil))
		if err := c.update(db, b, path, sum); err != nil {
			return err
		}
	}
	if cnt > 0 {
		c.Debugger(fmt.Sprintf("read %d items within the 7-Zip archive", cnt))
	}
	return nil
}

// Read opens the named archive, hashes and saves the content to the bucket.
func (c *Config) Read(db *bolt.DB, b parse.Bucket, name, mimeExt string) error {
	if db == nil {
		return bberr.ErrDatabaseNotOpen
	}
	c.Debugger("read archiver: " + name)
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
		printer.StderrCR(err)
		return nil
	}
	switch archive.Supported(f) {
	case true:
		w, ok := f.(archiver.Walker)
		if !ok {
			printer.StderrCR(fmt.Errorf("%w: %s: %s", archive.ErrType, lookup, name))
			return nil
		}
		cnt, err = c.readWalk(db, b, name, cnt, w)
		if err != nil {
			printer.Stderr(err)
		}
	default:
		color.Warn.Printf("Unsupported archive: '%s'\n", name)
		return nil
	}
	if cnt > 0 {
		c.Debugger(fmt.Sprintf("read %d items within the archive", cnt))
	}
	return nil
}

func (c *Config) walkDir(db *bolt.DB, root string, skip []string) error {
	if db == nil {
		return bberr.ErrDatabaseNotOpen
	}
	c.Debugger("walk directory: " + root)
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if c.Debug {
			s := "walk file"
			if d.IsDir() {
				s = "walk subdirectory"
			}
			c.Debugger(fmt.Sprintf("%s: %s", s, path))
		}
		if err != nil {
			if errors.Is(err, fs.ErrPermission) {
				return nil
			}
			c.Debugger(errRead + err.Error())
			return err
		}
		if path == root || skipSelf(path, skip...) {
			return c.walkDebug(" -skip self", nil)
		}
		if err := SkipFS(false, true, true, d); err != nil {
			return c.walkDebug(" -skip directory or system file", err)
		}
		c.Files++
		if err := c.walkCompare(db, root, path); err != nil {
			if errors.Is(err, ErrPathExist) {
				return nil
			}
			return err
		}
		_, _ = fmt.Fprint(os.Stdout, PrintWalk(false, c))
		if err := c.Checksum(db, path, root); err != nil {
			return err
		}
		return nil
	})
}

func (c *Config) statSources(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		c.Debugger(path)
		if err != nil {
			c.Debugger("stat sources error: " + err.Error())
			return err
		}
		if root == path {
			return nil
		}
		if err := SkipFS(true, false, true, d); err != nil {
			ignore(err)
			return nil
		}
		c.Sources = append(c.Sources, path)
		return nil
	})
}

// create a new, empty bucket in the database.
func (c *Config) create(db *bolt.DB, name parse.Bucket) error {
	if db == nil {
		return bberr.ErrDatabaseNotOpen
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
		return bberr.ErrDatabaseNotOpen
	}
	// use all the buckets if no specific buckets are provided
	if !c.Test && len(c.Buckets) == 0 {
		if err := c.SetAllBuckets(db); err != nil {
			return err
		}
	}
	// normalise bucket names
	for i, b := range c.Buckets {
		abs, err := database.Abs(string(b))
		if err != nil {
			printer.StderrCR(err)
			c.Buckets[i] = ""

			continue
		}
		c.Buckets[i] = parse.Bucket(abs)
	}
	if c.Compare == nil {
		c.Compare = make(parse.Checksums)
	}
	if !c.Test && c.Compare == nil {
		for i, b := range c.Buckets {
			_, err := c.SetCompares(db, b)
			if err != nil {
				return err
			}
			c.Debugger(fmt.Sprintf("init %d: %s", i, b))
		}
	}
	return nil
}

// lookup the checksum value in c.compare and return the file path.
func (c *Config) lookupOne(sum parse.Checksum) string {
	if len(c.Compare) == 0 {
		c.Compare = make(parse.Checksums)
	}
	c.Debugger(fmt.Sprintf("look up checksum in the compare data, %d items total: %x",
		len(c.Compare), sum))
	if f := c.Compare[sum]; f != "" {
		c.Debugger("lookupOne match: " + f)
		return f
	}
	return ""
}

// skipFiles returns the value of c.sources as strings.
func (c *Config) skipFiles() []string {
	var files []string
	files = append(files, c.Sources...)
	return files
}

// walkCompare walks the root directory and adds the checksums of the files to c.compare.
func (c *Config) walkCompare(db *bolt.DB, root, path string) error {
	if db == nil {
		return bberr.ErrDatabaseNotOpen
	}
	if c.Compare == nil {
		c.Compare = make(parse.Checksums)
	}
	return db.View(func(tx *bolt.Tx) error {
		if !c.Test && !c.Quiet && !c.Debug {
			_, _ = fmt.Fprint(os.Stdout, printer.Status(c.Files, -1, printer.Scan))
		}
		b := tx.Bucket([]byte(root))
		if b == nil {
			return ErrNoNamedBucket
		}
		h := b.Get([]byte(path))
		c.Debugger(fmt.Sprintf(" - %d/%d items: %x", len(c.Compare), c.Files, h))
		if len(h) > 0 {
			var sum parse.Checksum
			copy(sum[:], h)
			c.Compare[sum] = path
			return ErrPathExist
		}
		return nil
	})
}

func (c *Config) walkDebug(s string, err error) error {
	c.Debugger(s)
	return err
}

func (c *Config) readArchive(db *bolt.DB, b parse.Bucket, path string) error {
	if db == nil {
		return bberr.ErrDatabaseNotOpen
	}
	// detect archive type by file extension
	mimeExt := strings.ToLower(filepath.Ext(path))
	ok := (archive.MIME(path) != "")
	c.Debugger(fmt.Sprintf("is known extension: %v, %s", ok, mimeExt))
	if !ok {
		// detect archive type by mime type
		mime, err := archive.ReadMIME(path)
		if errors.Is(err, archive.ErrFilename) {
			if mime != "" {
				c.Debugger(fmt.Sprintf("archive not supported: %s: %s", mime, path))
			}
			return nil
		}
		if err != nil {
			return err
		}
		mimeExt = archive.Extension(mime)
	}
	c.Files++
	c.Debugger(fmt.Sprintf("walkCompare #%d", c.Files))
	if errD := c.walkCompare(db, string(b), path); errD != nil {
		if !errors.Is(errD, ErrPathExist) {
			printer.ErrFatal(errD)
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
	return slices.Contains(c.Sources, abs)
}

// listItems sets c.sources to list all the filenames used in the bucket.
func (c *Config) listItems(db *bolt.DB, bucket string) error {
	if db == nil {
		return bberr.ErrDatabaseNotOpen
	}
	c.Debugger("list bucket items: " + bucket)
	abs, err := database.AbsB(bucket)
	if err != nil {
		printer.StderrCR(err)
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
		if errors.Is(err, bberr.ErrBucketNotFound) {
			return fmt.Errorf("%w: '%s'", err, abs)
		}
		return err
	}
	return nil
}

func (c *Config) printer(w io.Writer, path string) error {
	sum, err := parse.Read(path)
	if err != nil {
		return err
	}
	l := c.lookupOne(sum)
	if l == "" {
		return ErrNoMatch
	}
	if l == path {
		return ErrNoMatch
	}
	printl(w, Match(path, l))
	return nil
}

func (c *Config) statSource() (bool, int, error) {
	name := c.GetSource()
	stat, err := os.Stat(name)
	if err != nil {
		c.Debugger("path is not found, but there is an absolute resolved path")
		name, err = filepath.Abs(name)
		if err != nil {
			return false, 0, os.ErrNotExist
		}
		stat, err = os.Stat(name)
		if err != nil {
			return false, 0, err
		}
		if err = c.SetSource(stat.Name()); err != nil {
			return false, 0, err
		}
	}
	if !stat.IsDir() {
		if stat.Size() == 0 {
			return false, 1, ErrFileEmpty
		}
		return false, 1, nil
	}
	if err := SkipDirs(name); err != nil {
		return false, 0, err
	}
	files := 0
	root := name
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		c.Debugger("counting: " + path)
		if err != nil {
			c.Debugger(errRead + err.Error())
			ignore(err)
			return nil
		}
		if root == path {
			return nil
		}
		if err := SkipFS(true, false, true, d); err != nil {
			ignore(err)
			return nil
		}
		files++
		return nil
	})
	if err != nil {
		return true, files, err
	}
	return true, files, nil
}

func (c *Config) walkBucket(b parse.Bucket, buckets int) (int, error) {
	root := string(b)
	stat, err := os.Stat(root)
	if err != nil {
		return 0, err
	}
	if !stat.IsDir() {
		return 0, ErrPathIsFile
	}
	if err := SkipDirs(root); err != nil {
		return 0, err
	}
	c.Debugger("walking bucket: " + root)
	return buckets, filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			c.Debugger(errRead + err.Error())
			return err
		}
		if root == path {
			return nil
		}
		c.Debugger("walking bucket item: " + path)
		if err := SkipFS(true, false, true, d); err != nil {
			ignore(err)
			return nil
		}
		buckets++
		return nil
	})
}

func (c *Config) readRecover(archive string) {
	if err := recover(); err != nil {
		if !c.Quiet {
			if !c.Debug {
				printl(os.Stdout)
			}
			color.Warn.Printf("Unsupported archive: '%s'\n", archive)
		}
		c.Debugger(fmt.Sprint(err))
	}
}

func (c *Config) readWalk(db *bolt.DB, b parse.Bucket, archive string, cnt int, w archiver.Walker) (int, error) {
	if db == nil {
		return -1, bberr.ErrDatabaseNotOpen
	}
	return cnt, w.Walk(archive, func(f archiver.File) error {
		notFile := f.IsDir() || !f.FileInfo.Mode().IsRegular()
		if notFile || SkipFile(f.Name()) {
			return nil
		}
		// Security: Prevent path traversal attacks from malicious archives
		fullPath := filepath.Join(archive, f.Name())
		cleanPath := filepath.Clean(fullPath)
		archiveDir := filepath.Clean(archive) + string(filepath.Separator)

		// Validate that the path doesn't escape the archive directory
		// Also check for absolute paths that bypass the archive directory
		if !strings.HasPrefix(cleanPath, archiveDir) || filepath.IsAbs(f.Name()) {
			return fmt.Errorf("path traversal attempt detected: %s", f.Name())
		}
		path := cleanPath
		if c.findItem(path) {
			return nil
		}
		buf, h := make([]byte, oneMb), sha256.New()
		if _, err := io.CopyBuffer(h, f, buf); err != nil {
			printer.Stderr(err)
			return nil
		}
		var sum parse.Checksum
		copy(sum[:], h.Sum(nil))
		if err := c.update(db, b, path, sum); err != nil {
			printer.Stderr(err)
		}
		cnt++
		return nil
	})
}

// update saves the checksum and path values to the bucket.
func (c *Config) update(db *bolt.DB, b parse.Bucket, path string, sum parse.Checksum) error {
	if db == nil {
		return bberr.ErrDatabaseNotOpen
	}
	if len(c.Compare) == 0 {
		c.Compare = make(parse.Checksums)
	}
	c.Debugger("update archiver: " + path)
	if err := db.Update(func(tx *bolt.Tx) error {
		b1 := tx.Bucket([]byte(b))
		if b1 == nil {
			return bberr.ErrBucketNotFound
		}
		return b1.Put([]byte(path), sum[:])
	}); err != nil {
		return err
	}
	c.Compare[sum] = path
	return nil
}
