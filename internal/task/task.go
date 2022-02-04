// © Ben Garrett https://github.com/bengarrett/dupers

// Package dupers is the blazing-fast file duplicate checker and filename search.
package task

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/bengarrett/dupers/database"
	"github.com/bengarrett/dupers/dupe"
	"github.com/bengarrett/dupers/internal/cmd"
	"github.com/bengarrett/dupers/internal/out"
	"github.com/dustin/go-humanize"
	"github.com/gookit/color"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

var (
	ErrCmd          = errors.New("command is unknown")
	ErrDatabaseName = errors.New("database has no bucket name")
	ErrImport       = errors.New("import filepath is missing")
	ErrNewName      = errors.New("a new directory is required")
)

const (
	dbf   = "database"
	dbs   = "db"
	dbk   = "backup"
	dcn   = "clean"
	dex   = "export"
	dim   = "import"
	dls   = "ls"
	dmv   = "mv"
	drm   = "rm"
	dup   = "up"
	dupp  = "up+"
	fhlp  = "-help"
	winOS = "windows"
)

// ChkWinDirs checks the arguments for invalid escaped quoted paths when using using Windows cmd.exe.
func ChkWinDirs() {
	if runtime.GOOS == winOS && len(flag.Args()) > 1 {
		for _, s := range flag.Args()[1:] {
			if err := cmd.ChkWinDir(s); err != nil {
				out.ErrFatal(err)
			}
		}
	}
}

// DatabaseCmd parses the database commands.
func DatabaseCmd(c *dupe.Config, quiet bool, args ...string) {
	checkDB()
	buckets := [2]string{}
	copy(buckets[:], args)

	switch args[0] {
	case dbk:
		backupDB(quiet)
	case dcn:
		cleanupDB(quiet, c.Debug)
	case dbs, dbf:
		s, err := database.Info()
		if err != nil {
			out.ErrCont(err)
		}
		fmt.Println(s)
	case dex:
		exportBucket(quiet, buckets)
	case dim:
		importBucket(quiet, buckets)
	case dls:
		listBucket(quiet, buckets)
	case dmv:
		buckets := [3]string{}
		copy(buckets[:], args)
		moveBucket(quiet, buckets)
	case drm:
		removeBucket(quiet, buckets)
	case dup:
		rescanBucket(c, false, buckets)
	case dupp:
		rescanBucket(c, true, buckets)
	default:
		out.ErrFatal(ErrCmd)
	}
}

// DupeCmd parses the dupe command.
func DupeCmd(c *dupe.Config, f *cmd.Flags, args ...string) {
	if c.Debug {
		s := fmt.Sprintf("dupeCmd: %s", strings.Join(args, " "))
		out.PBug(s)
	}
	l := len(args)
	if l == 1 {
		const minArgs = 2

		dupeCmdErr(l, 0, minArgs)
	}
	// fetch bucket info
	b, err := database.AllBuckets(nil)
	if err != nil {
		out.ErrFatal(err)
	}
	const minArgs = 3
	if l < minArgs && len(b) == 0 {
		dupeCmdErr(l, len(b), minArgs)
	}
	// directory or a file to match
	c.SetToCheck(args[1])
	// directories and files to scan, a bucket is the name given to database tables
	if buckets := args[2:]; len(buckets) == 0 {
		c.SetAllBuckets()
	} else {
		c.SetBuckets(buckets...)
		checkDupePaths(c)
	}
	if c.Debug {
		s := fmt.Sprintf("buckets: %s", c.PrintBuckets())
		out.PBug(s)
	}
	// files or directories to compare (these are not saved to database)
	if err := c.WalkSource(); err != nil {
		out.ErrFatal(err)
	}
	if c.Debug {
		out.PBug("walksource complete.")
	}
	// walk, scan and save file paths and hashes to the database
	dupeLookup(c, f)
	if !c.Quiet {
		fmt.Print(out.RMLine())
	}
	// print the found dupes
	fmt.Print(c.Print())
	// remove files
	dupeCleanup(c, f)
	// summaries
	if !c.Quiet {
		fmt.Println(c.Status())
	}
}

// checkBkt prints the missing bucket name error.
func checkBkt(term, cmd, name string) {
	if name != "" {
		return
	}
	out.ErrCont(ErrDatabaseName)
	fmt.Printf("Cannot %s the bucket as no bucket name was provided.\n", term)
	if cmd == dmv {
		out.Example(fmt.Sprintf("\ndupers %s <bucket name> <new directory>", cmd))
		out.ErrFatal(nil)
	}
	out.Example(fmt.Sprintf("\ndupers %s <bucket name>", cmd))
	out.ErrFatal(nil)
}

// checkDB checks the database file.
func checkDB() {
	path, err := database.DB()
	if err != nil {
		out.ErrFatal(err)
	}
	i, err1 := os.Stat(path)
	if os.IsNotExist(err1) {
		out.ErrCont(database.ErrDBNotFound)
		fmt.Printf("\n%s\nThe database will be located at: %s\n", database.NotFound, path)
		os.Exit(0)
	} else if err1 != nil {
		out.ErrFatal(err1)
	}
	if i.Size() == 0 {
		out.ErrCont(database.ErrDBZeroByte)
		s := "This error occures when dupers cannot save any data to the file system."
		fmt.Printf("\n%s\nThe database is located at: %s\n", s, path)
		os.Exit(1)
	}
}

// dupeCleanup runs the cleanup commands when the appropriate flags are set.
func dupeCleanup(c *dupe.Config, f *cmd.Flags) {
	if *f.Sensen {
		if c.Debug {
			out.PBug("remove all non unique Windows and MS-DOS files.")
		}
		fmt.Print(c.Remove())
		fmt.Print(c.RemoveAll())
		fmt.Print(c.Clean())
		return
	}
	if *f.Rm || *f.RmPlus {
		if c.Debug {
			out.PBug("remove duplicate files.")
		}
		fmt.Print(c.Remove())
		if *f.RmPlus {
			if c.Debug {
				out.PBug("remove empty directories.")
			}
			fmt.Print(c.Clean())
		}
	}
}

// dupeLookup cleans and updates buckets for changes on the file system.
func dupeLookup(c *dupe.Config, f *cmd.Flags) {
	if c.Debug {
		out.PBug("dupe lookup.")
	}
	// normalise bucket names
	for i, b := range c.Buckets() {
		abs, err := database.Abs(string(b))
		if err != nil {
			out.ErrCont(err)
			c.Buckets()[i] = ""

			continue
		}
		c.Buckets()[i] = dupe.Bucket(abs)
	}
	buckets := make([]string, 0, len(c.Buckets()))
	for _, b := range c.Buckets() {
		buckets = append(buckets, string(b))
	}
	if !*f.Lookup && len(buckets) > 0 {
		if c.Debug {
			out.PBug("non-fast mode, database cleanup.")
		}
		if err := database.Clean(c.Quiet, c.Debug, buckets...); err != nil {
			out.ErrCont(err)
		}
	}
	if *f.Lookup {
		if c.Debug {
			out.PBug("read the hash values in the buckets.")
		}
		fastErr := false
		for _, b := range c.Buckets() {
			if i := c.SetCompares(b); i > 0 {
				continue
			}
			fastErr = true
			fmt.Println("The -fast flag cannot be used for this dupe query")
		}
		if !fastErr {
			return
		}
	}
	if c.Debug {
		out.PBug("walk the buckets.")
	}
	c.WalkDirs()
}

// SearchCmd runs the search command.
func SearchCmd(f *cmd.Flags, args ...string) {
	l := len(args)
	searchCmdErr(l)
	term, buckets := args[1], []string{}
	const minArgs = 2
	if l > minArgs {
		buckets = args[minArgs:]
	}
	m := searchCompare(f, term, buckets)
	fmt.Print(dupe.Print(*f.Quiet, *f.Exact, term, m))
	if !*f.Quiet {
		l := 0
		if m != nil {
			l = len(*m)
		}
		fmt.Println(cmd.SearchSummary(l, term, *f.Exact, *f.Filename))
	}
}

func searchCompare(f *cmd.Flags, term string, buckets []string) *database.Matches {
	var err error
	var m *database.Matches
	switch {
	case *f.Filename && !*f.Exact:
		if m, err = database.CompareBaseNoCase(term, buckets...); err != nil {
			searchErr(err)
		}
	case *f.Filename && *f.Exact:
		if m, err = database.CompareBase(term, buckets...); err != nil {
			searchErr(err)
		}
	case !*f.Filename && !*f.Exact:
		if m, err = database.CompareNoCase(term, buckets...); err != nil {
			searchErr(err)
		}
	case !*f.Filename && *f.Exact:
		if m, err = database.Compare(term, buckets...); err != nil {
			searchErr(err)
		}
	}
	return m
}

// exportBucket saves the bucket to a csv file.
func exportBucket(quiet bool, args [2]string) {
	checkBkt(dex, dex, args[1])
	name, err := database.Abs(args[1])
	if err != nil {
		out.ErrFatal(err)
	}
	if errEx := database.Exist(name, nil); errors.Is(errEx, database.ErrBucketNotFound) {
		out.ErrCont(errEx)
		fmt.Printf("Bucket name: %s\n", name)
		out.Example("\ndupers export <bucket name>")
		out.ErrFatal(nil)
	} else if errEx != nil {
		out.ErrFatal(errEx)
	}
	exp, errEx := database.ExportCSV(name, nil)
	if errEx != nil {
		out.ErrFatal(errEx)
	}
	s := fmt.Sprintf("%s %s\n", color.Secondary.Sprint("Bucket name:"), color.Debug.Sprint(name))
	s += fmt.Sprintf("The exported bucket file is at: %s", exp)
	out.Response(s, quiet)
}

// importBucket saves a csv file to the database.
func importBucket(quiet bool, args [2]string) {
	if args[1] == "" {
		out.ErrCont(ErrImport)
		fmt.Println("Cannot import file as no filepath was provided.")
		out.Example(fmt.Sprintf("\ndupers %s <filepath>", dim))
		out.ErrFatal(nil)
	}
	name, err := database.Abs(args[1])
	if err != nil {
		out.ErrFatal(err)
	}
	r, errIm := database.ImportCSV(name, nil)
	if errIm != nil {
		out.ErrFatal(errIm)
	}
	p := message.NewPrinter(language.English)
	s := p.Sprintf("\rSuccessfully imported %d records.", number.Decimal(r))
	out.Response(s, quiet)
}

// listBucket lists the content of a bucket to the stdout.
func listBucket(quiet bool, args [2]string) {
	checkBkt("list", dls, args[1])
	name, err := database.Abs(args[1])
	if err != nil {
		out.ErrFatal(err)
	}
	ls, err := database.List(name, nil)
	if err != nil {
		out.ErrCont(err)
	}
	// sort the filenames
	names := make([]string, 0, len(ls))
	for name := range ls {
		names = append(names, string(name))
	}
	sort.Strings(names)
	for _, name := range names {
		sum := ls[database.Filepath(name)]
		fmt.Printf("%x %s\n", sum, name)
	}
	if cnt := len(ls); !quiet && cnt > 0 {
		p := message.NewPrinter(language.English)
		fmt.Printf("%s %s\n", color.Primary.Sprint(p.Sprint(number.Decimal(cnt))),
			color.Secondary.Sprint("items listed. Checksums are 32 byte, SHA-256 (FIPS 180-4)."))
	}
}

// moveBucket renames a bucket by duplicating it to a new bucket location.
func moveBucket(quiet bool, args [3]string) {
	b, dir := args[1], args[2]
	checkBkt("move and rename", dmv, b)
	name, err := database.Abs(b)
	if err != nil {
		out.ErrFatal(err)
	}
	if errEx := database.Exist(name, nil); errors.Is(errEx, database.ErrBucketNotFound) {
		out.ErrCont(errEx)
		fmt.Printf("Bucket name: %s\n", name)
		out.Example("\ndupers mv <bucket name> <new directory>")
		out.ErrFatal(nil)
	} else if errEx != nil {
		out.ErrFatal(errEx)
	}
	if dir == "" {
		out.ErrCont(ErrNewName)
		fmt.Println("Cannot move bucket within the database as no new directory was provided.")
		out.Example(fmt.Sprintf("\ndupers mv %s <new directory>", b))
		out.ErrFatal(nil)
	}
	newName, err := database.Abs(dir)
	if err != nil {
		out.ErrFatal(err)
	}
	if newName == "" {
		out.ErrFatal(ErrNewName)
	}
	if !quiet {
		fmt.Printf("%s\t%s\n%s\t%s\n",
			color.Secondary.Sprint("Bucket name:"), color.Debug.Sprint(name),
			"New name:", color.Debug.Sprint(newName))
		fmt.Println("Renames the database bucket, but this does not make changes to the file system.")
		if !out.YN("Rename bucket", out.No) {
			return
		}
	}
	if err := database.Rename(name, newName); err != nil {
		out.ErrFatal(err)
	}
}

// removeBucket removes the bucket from the database.
func removeBucket(quiet bool, args [2]string) {
	checkBkt("remove", drm, args[1])
	name, err := database.Abs(args[1])
	if err != nil {
		out.ErrFatal(err)
	}

	items, err := database.Count(name, nil)
	if errors.Is(err, database.ErrBucketNotFound) {
		// fallback Abs check
		name, err = filepath.Abs(args[1])
		if err != nil {
			out.ErrFatal(err)
		}
		items, err = database.Count(name, nil)
		if errors.Is(err, database.ErrBucketNotFound) {
			bucketNoFound(name, err)
			return
		}
	}
	if !quiet {
		fmt.Printf("%s\t%s\n", color.Secondary.Sprint("Bucket:"), color.Debug.Sprint(name))
		p := message.NewPrinter(language.English)
		fmt.Printf("%s\t%s\n", color.Secondary.Sprint("Items:"), color.Debug.Sprint(p.Sprint(items)))
		if !out.YN("Remove this bucket", out.No) {
			return
		}
	}
	rmBucket(name, args[1])
	s := fmt.Sprintf("Removed bucket from the database: '%s'\n", name)
	out.Response(s, quiet)
}

func rmBucket(name, retry string) {
	err := database.RM(name)
	if err == nil {
		return
	}
	if errors.Is(err, database.ErrBucketNotFound) {
		// retry with the original argument
		if err1 := database.RM(retry); err1 != nil {
			if errors.Is(err1, database.ErrBucketNotFound) {
				bucketNoFound(name, err1)
			}
			out.ErrFatal(err1)
		}
	}
}

func bucketNoFound(name string, err error) {
	out.ErrCont(err)
	fmt.Printf("Bucket to remove: %s\n", color.Danger.Sprint(name))
	buckets, err2 := database.AllBuckets(nil)
	if err2 != nil {
		out.ErrFatal(err2)
	}
	if len(buckets) == 0 {
		fmt.Println("There are no buckets in the database")
		out.ErrFatal(nil)
	}
	fmt.Printf("Buckets in use:   %s\n", strings.Join(buckets, "\n\t\t  "))
	out.ErrFatal(nil)
}

// rescanBucket rescans the bucket for any changes on the file system.
func rescanBucket(c *dupe.Config, plus bool, args [2]string) {
	cmd := dup
	if plus {
		cmd = dupp
	}
	checkBkt("add or update", cmd, args[1])
	path, err := database.Abs(args[1])
	if err != nil {
		out.ErrFatal(err)
	}
	name := dupe.Bucket(path)
	if plus {
		if err := c.WalkArchiver(name); err != nil {
			out.ErrFatal(err)
		}
	} else if err := c.WalkDir(name); err != nil {
		out.ErrFatal(err)
	}
	if !c.Quiet {
		fmt.Println(c.Status())
	}
}

// backupDB saves the database to a binary file.
func backupDB(quiet bool) {
	n, w, err := database.Backup()
	if err != nil {
		out.ErrFatal(err)
	}
	s := fmt.Sprintf("A new copy of the database (%s) is at: %s", humanize.Bytes(uint64(w)), n)
	out.Response(s, quiet)
}

// cleanupDB cleans and compacts the database.
func cleanupDB(quiet, debug bool) {
	if err := database.Clean(quiet, debug); err != nil {
		if b := errors.Is(err, database.ErrDBClean); !b {
			out.ErrFatal(err)
		}
		out.ErrCont(err)
	}
	if err := database.Compact(debug); err != nil {
		if b := errors.Is(err, database.ErrDBCompact); !b {
			out.ErrFatal(err)
		}
	}
}
