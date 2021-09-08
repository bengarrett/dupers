// © Ben Garrett https://github.com/bengarrett/dupers

// Package dupers is the blazing-fast file duplicate checker and filename search.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/bengarrett/dupers/database"
	"github.com/bengarrett/dupers/dupers"
	"github.com/bengarrett/dupers/out"
	"github.com/dustin/go-humanize"
	"github.com/gookit/color"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

// checkBkt prints the missing bucket name error.
func checkBkt(term, cmd, name string) {
	if name != "" {
		return
	}
	out.ErrCont(ErrDatabaseName)
	fmt.Printf("Cannot %s the bucket as no bucket name was provided.\n", term)
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

// chkWinDirs checks the arguments for invalid escaped quoted paths when using using Windows cmd.exe.
func chkWinDirs() {
	if runtime.GOOS == winOS && len(flag.Args()) > 1 {
		for _, s := range flag.Args()[1:] {
			if err := chkWinDir(s); err != nil {
				out.ErrFatal(err)
			}
		}
	}
}

// chkWinDir checks the string for invalid escaped quoted paths when using using Windows cmd.exe.
func chkWinDir(s string) error {
	if s == "" {
		return nil
	}
	const dblQuote rune = 34
	r := []rune(s)
	l := len(r)
	first, last := r[0:1][0], r[l-1 : l][0]
	if first == dblQuote && last == dblQuote {
		return nil // okay as the string is fully quoted
	}
	if first != dblQuote && last != dblQuote {
		return nil // okay as the string is not quoted
	}
	// otherwise there is a problem, as only the start or end of the string is quoted.
	// this is caused by flag.Parse() treating the \" prefix on a quoted directory path as an escaped quote.
	// so "C:\Example\" will be incorrectly parsed as C:\Example"
	w := new(bytes.Buffer)
	fmt.Fprint(w, "please remove the trailing backslash \\ character from any quoted directory paths")
	if usr, err := os.UserHomeDir(); err == nil {
		fmt.Fprint(w, "\n")
		fmt.Fprint(w, color.Success.Sprint("Good: "))
		fmt.Fprintf(w, "\"%s\" ", usr)
		fmt.Fprint(w, "\n")
		fmt.Fprint(w, color.Warn.Sprint("Bad: "))
		fmt.Fprintf(w, "\"%s\\\"", usr)
	}
	return fmt.Errorf("%w\n%s", ErrWindowsDir, w.String())
}

// databaseCmd parses the database commands.
func databaseCmd(c *dupers.Config, quiet bool, args ...string) {
	checkDB()
	arr := [2]string{}
	copy(arr[:], args)
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
		exportBucket(quiet, arr)
	case dim:
		importBucket(quiet, arr)
	case dls:
		listBucket(quiet, arr)
	case dmv:
		arr := [3]string{}
		copy(arr[:], args)
		moveBucket(quiet, arr)
	case drm:
		removeBucket(quiet, arr)
	case dup:
		rescanBucket(c, false, arr)
	case dupp:
		rescanBucket(c, true, arr)
	default:
		out.ErrFatal(ErrCmd)
	}
}

// dupeCmd parses the dupe command.
func dupeCmd(c *dupers.Config, f *cmdFlags, args ...string) {
	if c.Debug {
		s := fmt.Sprintf("dupeCmd: %s", strings.Join(args, " "))
		out.Bug(s)
	}
	l := len(args)
	// fetch bucket info
	b, err := database.AllBuckets(nil)
	if err != nil {
		out.ErrFatal(err)
	}
	const minArgs = 3
	if l < minArgs && len(b) == 0 {
		dupeCmdErr(l, len(b))
	}
	// directory or a file to match
	c.SetToCheck(args[1])
	// directories and files to scan, a bucket is the name given to database tables
	arr := args[2:]
	c.SetBuckets(arr...)
	if arr == nil {
		c.SetAllBuckets()
	}
	if c.Debug {
		s := fmt.Sprintf("buckets: %s", c.PrintBuckets())
		out.Bug(s)
	}
	// check the user supplied directories
	checkDupePaths(c)
	// files or directories to compare (these are not saved to database)
	if err := c.WalkSource(); err != nil {
		out.ErrFatal(err)
	}
	if c.Debug {
		out.Bug("walksource complete.")
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

// dupeCleanup runs the cleanup commands when the appropriate flags are set.
func dupeCleanup(c *dupers.Config, f *cmdFlags) {
	if *f.sensen {
		if c.Debug {
			out.Bug("remove all non unique Windows and MS-DOS files.")
		}
		fmt.Print(c.Remove())
		fmt.Print(c.RemoveAll())
		fmt.Print(c.Clean())
		return
	}
	if *f.rm || *f.rmPlus {
		if c.Debug {
			out.Bug("remove duplicate files.")
		}
		fmt.Print(c.Remove())
		if *f.rmPlus {
			if c.Debug {
				out.Bug("remove empty directories.")
			}
			fmt.Print(c.Clean())
		}
	}
}

// dupeLookup cleans and updates buckets for changes on the file system.
func dupeLookup(c *dupers.Config, f *cmdFlags) {
	if c.Debug {
		out.Bug("dupe lookup.")
	}
	var bkts []string
	for _, b := range c.Buckets() {
		bkts = append(bkts, string(b))
	}
	if !*f.lookup && len(bkts) > 0 {
		if c.Debug {
			out.Bug("non-fast mode, database cleanup.")
		}
		if err := database.Clean(c.Quiet, c.Debug, bkts...); err != nil {
			out.ErrCont(err)
		}
	}
	if *f.lookup {
		if c.Debug {
			out.Bug("read the hash values in the buckets.")
		}
		for _, b := range c.Buckets() {
			c.SetCompares(b)
		}
		return
	}
	if c.Debug {
		out.Bug("walk the buckets.")
	}
	c.WalkDirs()
}

// searchCmd runs the search command.
func searchCmd(f *cmdFlags, args ...string) {
	l := len(args)
	searchCmdErr(l)
	term := args[1]
	var (
		buckets = []string{}
		m       *database.Matches
		err     error
	)
	const minArgs = 2
	if l > minArgs {
		buckets = args[2:]
	}
	if *f.filename {
		if !*f.exact {
			if m, err = database.CompareBaseNoCase(term, buckets...); err != nil {
				searchErr(err)
			}
		}
		if *f.exact {
			if m, err = database.CompareBase(term, buckets...); err != nil {
				searchErr(err)
			}
		}
	}
	if !*f.filename {
		if !*f.exact {
			if m, err = database.CompareNoCase(term, buckets...); err != nil {
				searchErr(err)
			}
		}
		if *f.exact {
			if m, err = database.Compare(term, buckets...); err != nil {
				searchErr(err)
			}
		}
	}
	fmt.Print(dupers.Print(*f.quiet, m))
	if !*f.quiet {
		l := 0
		if m != nil {
			l = len(*m)
		}
		fmt.Println(searchCmdSummary(l, term, *f.exact, *f.filename))
	}
}

// searchCmdSummary formats the results of the search command.
func searchCmdSummary(total int, term string, exact, filename bool) string {
	str := func(t, s, term string) string {
		return fmt.Sprintf("%s%s exist for '%s'.", t, color.Secondary.Sprint(s), color.Bold.Sprint(term))
	}
	s, r := "", "results"
	if total == 0 {
		return fmt.Sprintf("No results exist for '%s'.", term)
	}
	if total == 1 {
		r = "result"
	}
	t := color.Primary.Sprint(total)
	if exact && filename {
		s += fmt.Sprintf(" exact filename %s", r)
		return str(t, s, term)
	}
	if exact {
		s += fmt.Sprintf(" exact %s", r)
		return str(t, s, term)
	}
	if filename {
		s += fmt.Sprintf(" filename %s", r)
		return str(t, s, term)
	}
	s += fmt.Sprintf(" %s", r)
	return str(t, s, term)
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
	f := args[1]
	if f == "" {
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
	var names []string
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
		fmt.Println("This renames the database bucket but it does not edit your file system.")
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
	if err := database.RM(name); err != nil {
		if errors.Is(err, database.ErrBucketNotFound) {
			// retry with the original argument
			if err1 := database.RM(args[1]); err1 != nil {
				if errors.Is(err1, database.ErrBucketNotFound) {
					bucketNoFound(name, err1)
				}
				out.ErrFatal(err1)
			}
		}
	}
	s := fmt.Sprintf("Removed bucket from the database: '%s'\n", name)
	out.Response(s, quiet)
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
func rescanBucket(c *dupers.Config, plus bool, args [2]string) {
	cmd := dup
	if plus {
		cmd = dupp
	}
	checkBkt("add or update", cmd, args[1])
	path, err := database.Abs(args[1])
	if err != nil {
		out.ErrFatal(err)
	}
	name := dupers.Bucket(path)
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
