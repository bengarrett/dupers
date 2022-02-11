// © Ben Garrett https://github.com/bengarrett/dupers
package bucket

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bengarrett/dupers/database"
	"github.com/bengarrett/dupers/dupe"
	"github.com/bengarrett/dupers/internal/out"
	"github.com/gookit/color"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

var (
	ErrDatabaseName = errors.New("database has no bucket name")
	ErrImport       = errors.New("import filepath is missing")
	ErrNewName      = errors.New("a new directory is required")
)

const (
	dmv  = "mv"
	dim  = "import"
	dls  = "ls"
	drm  = "rm"
	dup  = "up"
	dupp = "up+"
)

// Check prints the missing bucket name error.
func Check(term, cmd, name string) {
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

// Export the bucket as a CSV file.
func Export(quiet bool, args [2]string) {
	const x = "export"
	Check(x, x, args[1])
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
	exp, errEx := database.CSVExport(name, nil)
	if errEx != nil {
		out.ErrFatal(errEx)
	}
	s := fmt.Sprintf("%s %s\n", color.Secondary.Sprint("Bucket name:"), color.Debug.Sprint(name))
	s += fmt.Sprintf("The exported bucket file is at: %s", exp)
	out.Response(s, quiet)
}

// Import a CSV file into the database.
func Import(quiet bool, args [2]string) {
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
	r, errIm := database.CSVImport(name, nil)
	if errIm != nil {
		out.ErrFatal(errIm)
	}
	p := message.NewPrinter(language.English)
	s := p.Sprintf("\rSuccessfully imported %d records.", number.Decimal(r))
	out.Response(s, quiet)
}

// List the content of a bucket to the stdout.
func List(quiet bool, args [2]string) {
	Check("list", dls, args[1])
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

// Move renames a bucket by duplicating it to a new bucket location.
func Move(quiet bool, args [3]string) {
	b, dir := args[1], args[2]
	Check("move and rename", dmv, b)
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

// Remove the bucket from the database.
func Remove(quiet bool, args [2]string) {
	Check("remove", drm, args[1])
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
			notFound(name, err)
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
				notFound(name, err1)
			}
			out.ErrFatal(err1)
		}
	}
}

func notFound(name string, err error) {
	out.ErrCont(err)
	fmt.Printf("Bucket to remove: %s\n", color.Danger.Sprint(name))
	buckets, err2 := database.All(nil)
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

// Rescan the bucket for changes with the file system.
func Rescan(c *dupe.Config, plus bool, args [2]string) {
	cmd := dup
	if plus {
		cmd = dupp
	}
	Check("add or update", cmd, args[1])
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
