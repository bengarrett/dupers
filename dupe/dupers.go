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
	"sync"
	"time"

	"github.com/bengarrett/dupers/database"
	"github.com/bengarrett/dupers/dupe/internal/archive"
	"github.com/bengarrett/dupers/dupe/internal/parse"
	"github.com/bengarrett/dupers/internal/out"
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
	ErrNoBucket = errors.New("a named bucket is required")

	ErrPathIsFile  = errors.New("path is a file")
	ErrPathExist   = errors.New("path exists in the database bucket")
	ErrPathNoFound = errors.New("path does not exist")
)

// Config options.
type Config struct {
	Debug bool // Debug spams technobabble to stdout.
	Quiet bool // Quiet the feedback sent to stdout.
	Test  bool // Test toggles the internal unit test mode.
	parse.Parser
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
	dupeItem := c.ToCheck()
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
	files = c.walkPath(dupeItem)
	for _, b := range c.All() {
		var err error
		buckets, err = c.walkBucket(b, files, buckets)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			c.DPrint(err.Error())
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
			out.ErrFatal(err)
			return nil
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

func (c *Config) walkPath(root string) int {
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
		if c.Debug {
			out.PBug(err.Error())
		}
	}
	return checkCnt
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
		c.DPrint("path is a file")
		return name, ErrPathIsFile
	}
	return name, nil
}

// Checksum the named file and save it to the bucket.
func (c *Config) Checksum(name, bucket string) error {
	if c.Debug {
		out.PBug("update: " + name)
	}
	if c.DB == nil {
		return bolt.ErrDatabaseNotOpen
	}
	// read file, exit if it fails
	sum, err := parse.Read(name)
	if err != nil {
		return err
	}
	if sum == [32]byte{} {
		return nil
	}
	if err = c.DB.Update(func(tx *bolt.Tx) error {
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
		out.PBug("print duplicate results")
		s := fmt.Sprintf("comparing %d sources against %d unique items to compare",
			len(c.Sources), len(c.Compare))
		out.PBug(s)
	}
	w := new(bytes.Buffer)
	finds := 0
	for _, path := range c.Sources {
		sum, err := parse.Read(path)
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

// Remove duplicate files from the source directory.
func (c *Config) Remove() string {
	w := new(bytes.Buffer)
	if len(c.Sources) == 0 || len(c.Compare) == 0 {
		fmt.Fprintln(w, "No duplicate files to remove.          ")
		return w.String()
	}
	fmt.Fprintln(w)
	for _, path := range c.Sources {
		if c.Debug {
			out.PBug("remove read: " + path)
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if c.Debug {
				out.PBug("path is not exist: " + path)
			}

			continue
		}
		h, err := parse.Read(path)
		if err != nil {
			out.ErrCont(err)
		}
		if l := c.lookupOne(h); l == "" {
			continue
		}
		if c.Debug {
			out.PBug("remove delete: " + path)
		}
		err = os.Remove(path)
		fmt.Fprintln(w, printRM(path, err))
	}
	return w.String()
}

// Removes the directories from the source that do not contain unique MS-DOS or Windows programs.
func (c *Config) Removes() string {
	root := c.ToCheck()
	if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
		e := fmt.Errorf("%w: %s", ErrPathNoFound, root)
		out.ErrFatal(e)
	} else if err != nil {
		out.ErrFatal(err)
	}
	if len(c.Sources) == 0 {
		return ""
	}
	files, err := os.ReadDir(root)
	if err != nil {
		out.ErrCont(err)
	}
	if !c.Test {
		w := os.Stdout
		fmt.Fprintf(w, "%s %s\n", color.Secondary.Sprint("Target directory:"), color.Debug.Sprint(root))
		fmt.Fprintln(w, "Delete everything in the target directory, except for directories"+
			"\ncontaining unique Windows or MS-DOS programs and assets?")
		if input := out.YN("Please confirm", out.Nil); !input {
			os.Exit(0)
		}
		fmt.Fprintln(w)
	}
	return removes(root, files)
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
func (c *Config) WalkDirs() {
	c.init()
	if !c.Test && c.DB == nil {
		c.OpenWrite()
		defer c.DB.Close()
	}
	// walk through the directories provided
	for _, bucket := range c.All() {
		s := string(bucket)
		if c.Debug {
			out.PBug("walkdir bucket: " + s)
		}
		if err := c.WalkDir(bucket); err != nil {
			if errors.Is(errors.Unwrap(err), ErrPathNoFound) &&
				errors.Is(database.Exist(s, c.DB), database.ErrBucketNotFound) {
				out.ErrCont(err)
				continue
			}
			out.ErrCont(err)
		}
	}
	// handle any items that exist in the database but not in the file system
	// this would include items added using the `up+` archive scan command
	for _, b := range c.All() {
		if _, err := c.SetCompares(b); err != nil {
			out.ErrCont(err)
		}
	}
}

// WalkDir walks the named bucket directory for any new files to add their checksums to the database.
func (c *Config) WalkDir(name parse.Bucket) error {
	if name == "" {
		return ErrNoBucket
	}
	root := string(name)
	c.init()
	skip := c.skipFiles()
	// open database
	if !c.Test && c.DB == nil {
		c.OpenWrite()
		defer c.DB.Close()
	}
	// create a new bucket if needed
	if err := c.create(name); err != nil {
		return err
	}
	// walk the root directory
	return c.walkDir(root, skip)
}

func (c *Config) walkDir(root string, skip []string) error {
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
		if err1 := skipDir(d); err1 != nil {
			return c.walkDirSkip(" - skipping directory", err1)
		}
		if skipFile(d.Name()) {
			return c.walkDirSkip(" - skipping file", nil)
		}
		if !d.Type().IsRegular() {
			return c.walkDirSkip(" - skipping not regular file", nil)
		}
		if skipSelf(path, skip...) {
			return c.walkDirSkip(" - skipping self item", nil)
		}
		c.Files++
		if errW := walkCompare(root, path, c); errW != nil {
			if errors.Is(errW, ErrPathExist) {
				return nil
			}
			out.ErrFatal(errW)
		}
		fmt.Fprint(os.Stdout, PrintWalk(false, c))
		if err := c.Checksum(path, root); err != nil {
			out.ErrCont(err)
		}
		return err
	})
}

func (c *Config) walkDirSkip(s string, err error) error {
	if c.Debug {
		out.PBug(s)
	}
	return err
}

// WalkSource walks the source directory or a file to collect the hashed content for a future comparison.
func (c *Config) WalkSource() error {
	root := c.ToCheck()
	if c.Debug {
		out.PBug("walksource to check: " + root)
	}
	stat, err := os.Stat(root)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: %s", ErrPathNoFound, root)
	} else if err != nil {
		return fmt.Errorf("%w: %s", err, root)
	}
	if !stat.IsDir() {
		c.Sources = append(c.Sources, root)
		if c.Debug {
			out.PBug("items dupe check: " + strings.Join(c.Sources, " "))
		}
		return nil
	}
	if err := c.walkSource(root); err != nil {
		out.ErrCont(fmt.Errorf("item has a problem: %w", err))
		return nil
	}
	if c.Debug {
		out.PBug("directories dupe check: " + strings.Join(c.Sources, " "))
	}
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
func (c *Config) create(name parse.Bucket) error {
	if _, err := os.Stat(string(name)); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: %s", ErrPathNoFound, name)
	} else if err != nil {
		return err
	}
	if c.DB == nil {
		return bolt.ErrDatabaseNotOpen
	}
	return c.DB.Update(func(tx *bolt.Tx) error {
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
	if !c.Test && len(c.All()) == 0 {
		if err := c.SetAllBuckets(); err != nil {
			out.ErrFatal(err)
		}
	}
	// normalise bucket names
	for i, b := range c.All() {
		abs, err := database.Abs(string(b))
		if err != nil {
			out.ErrCont(err)
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
			_, _ = c.SetCompares(b)
			if c.Debug {
				s := fmt.Sprintf("init %d: %s", i, b)
				out.PBug(s)
			}
		}
	}
}

// lookup the checksum value in c.compare and return the file path.
func (c *Config) lookupOne(sum parse.Checksum) string {
	if c.Debug {
		s := fmt.Sprintf("look up checksum in the compare data, %d items total: %x", len(c.Compare), sum)
		out.PBug(s)
	}
	if f := c.Compare[sum]; f != "" {
		if c.Debug {
			out.PBug("lookupOne match: " + f)
		}
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
		out.ErrCont(e)
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
		if parse.Executable(path) {
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
func walkCompare(root, path string, c *Config) error {
	if c.DB == nil {
		return bolt.ErrDatabaseNotOpen
	}
	if c.Compare == nil {
		c.Compare = make(parse.Checksums)
	}
	return c.DB.View(func(tx *bolt.Tx) error {
		if !c.Test && !c.Quiet && !c.Debug {
			fmt.Fprint(os.Stdout, out.Status(c.Files, -1, out.Scan))
		}
		b := tx.Bucket([]byte(root))
		if b == nil {
			return ErrNoBucket
		}
		h := b.Get([]byte(path))
		if c.Debug {
			out.PBug(fmt.Sprintf(" - %d/%d items: %x", len(c.Compare), c.Files, h))
		}
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

// Bucket returns s as the Bucket type.
func Bucket(s string) parse.Bucket {
	return parse.Bucket(s)
}

// WalkArchiver walks the bucket directory saving the checksums of new files to the database.
// Any archived files supported by archiver will also have its content hashed.
// Archives within archives are currently left unwalked.
func (c *Config) WalkArchiver(name parse.Bucket) error {
	if name == "" {
		return ErrNoBucket
	}
	root := string(name)

	c.init()
	skip := c.skipFiles()
	if c.DB == nil {
		c.OpenWrite()
		defer c.DB.Close()
	}
	// create a new bucket if needed
	if err := c.create(name); err != nil {
		return err
	}
	// get a list of all the bucket filenames
	if err := c.listItems(root); err != nil {
		out.ErrCont(err)
	}
	// walk the root directory of the bucket
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
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
		err = c.walkThread(root, path, nil)
		if err != nil {
			out.ErrCont(err)
		}
		return nil
	})
	return err
}

func (c *Config) walkThread(bucket, path string, wg *sync.WaitGroup) error {
	// detect archive type by file extension
	mimeExt := strings.ToLower(filepath.Ext(path))
	ok := (archive.MIME(path) != "")
	if c.Debug {
		out.PBug(fmt.Sprintf("is known extension: %v, %s", ok, mimeExt))
	}
	if !ok {
		// detect archive type by mime type
		mime, err := archive.ReadMIME(path)
		if errors.Is(err, archive.ErrFilename) {
			if c.Debug && mime != "" {
				s := fmt.Sprintf("archive not supported: %s: %s", mime, path)
				out.PBug(s)
			}
			return nil
		} else if err != nil {
			return err
		}
		mimeExt = archive.Extension(mime)
	}
	c.Files++
	if c.Debug {
		s := fmt.Sprintf("walkCompare #%d", c.Files)
		out.PBug(s)
	}
	if errD := walkCompare(bucket, path, c); errD != nil {
		if !errors.Is(errD, ErrPathExist) {
			out.ErrFatal(errD)
		}
	}
	// multithread archive reader
	//wg.Add(1)

	go func() {
		switch mimeExt {
		case "":
			// not a supported archive, do nothing
		case archive.Ext7z:
			c.Read7Zip(bucket, path)
		default:
			c.Read(bucket, path, mimeExt)
		}
		//wg.Done()
	}()
	//wg.Wait()
	return nil
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
func (c *Config) listItems(bucket string) error {
	if c.Debug {
		out.PBug("list bucket items: " + bucket)
	}
	abs, err := database.AbsB(bucket)
	if err != nil {
		out.ErrCont(err)
	}
	if err = c.DB.View(func(tx *bolt.Tx) error {
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
	}); errors.Is(err, bolt.ErrBucketNotFound) {
		return fmt.Errorf("%w: '%s'", err, abs)
	} else if err != nil {
		return err
	}
	return nil
}

// Read7Zip opens the named 7-Zip archive, hashes and saves the content to the bucket.
func (c *Config) Read7Zip(bucket, name string) {
	if c.Debug {
		out.PBug("read 7zip: " + name)
	}
	r, err := sevenzip.OpenReader(name)
	if err != nil {
		out.ErrAppend(err)
		return
	}
	defer r.Close()
	cnt := 0
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		fp := filepath.Join(name, f.Name)
		if c.findItem(fp) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			out.ErrAppend(err)
			continue
		}
		defer rc.Close()
		cnt++
		buf, h := make([]byte, oneMb), sha256.New()
		if _, err := io.CopyBuffer(h, rc, buf); err != nil {
			out.ErrAppend(err)
			continue
		}
		var sum parse.Checksum

		copy(sum[:], h.Sum(nil))
		if err := c.update(fp, bucket, sum); err != nil {
			out.ErrAppend(err)
			continue
		}
	}
	if c.Debug && cnt > 0 {
		s := fmt.Sprintf("read %d items within the 7-Zip archive", cnt)
		out.PBug(s)
	}
}

// Read opens the named archive, hashes and saves the content to the bucket.
func (c *Config) Read(bucket, name, mimeExt string) {
	if c.Debug {
		out.PBug("read archiver: " + name)
	}
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
		out.ErrCont(err)
		return
	}
	switch archive.Supported(f) {
	case true:
		w, ok := f.(archiver.Walker)
		if !ok {
			out.ErrCont(fmt.Errorf("%w: %s: %s", archive.ErrType, lookup, name))
			return
		}
		cnt, err = c.readWalk(name, bucket, cnt, w)
		if err != nil {
			out.ErrAppend(err)
		}
	default:
		color.Warn.Printf("Unsupported archive: '%s'\n", name)
		return
	}
	if c.Debug && cnt > 0 {
		s := fmt.Sprintf("read %d items within the archive", cnt)
		out.PBug(s)
	}
}

func (c *Config) readWalk(archive, bucket string, cnt int, w archiver.Walker) (int, error) {
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
		fp := filepath.Join(archive, f.Name())
		if c.findItem(fp) {
			return nil
		}
		buf, h := make([]byte, oneMb), sha256.New()
		if _, err := io.CopyBuffer(h, f, buf); err != nil {
			out.ErrAppend(err)
			return nil
		}
		var sum parse.Checksum
		copy(sum[:], h.Sum(nil))
		if err := c.update(fp, bucket, sum); err != nil {
			out.ErrAppend(err)
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
		if c.Debug {
			out.PBug(fmt.Sprint(err))
		}
	}
}

// update saves the checksum and path values to the bucket.
func (c *Config) update(path, bucket string, sum parse.Checksum) error {
	if c.Debug {
		out.PBug("update archiver: " + path)
	}
	if c.DB == nil {
		return bolt.ErrDatabaseNotOpen
	}
	if err := c.DB.Update(func(tx *bolt.Tx) error {
		b1 := tx.Bucket([]byte(bucket))
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
