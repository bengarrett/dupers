// © Ben Garrett https://github.com/bengarrett/dupers
package bucket

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/bengarrett/dupers/internal/out"
	"github.com/bengarrett/dupers/pkg/database"
	"github.com/bengarrett/dupers/pkg/dupe"
	"github.com/gookit/color"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

var (
	ErrDatabaseName = errors.New("database has no bucket name")
	ErrImport       = errors.New("import filepath is missing")
	ErrNewName      = errors.New("a new directory is required")
	ErrBucketNil    = errors.New("bucket cannot be empty")
	ErrBucketEmpty  = errors.New("bucket contains no files")
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
// If an error exists an exit code will be returned.
func Check(term, cmd, name string) int {
	if name != "" {
		return 0
	}
	out.StderrCR(ErrDatabaseName)
	fmt.Fprintf(os.Stderr, "Cannot %s the bucket as no bucket name was provided.\n", term)
	if cmd == dmv {
		out.Example(fmt.Sprintf("\ndupers %s <bucket name> <new directory>", cmd))
		return 1
	}
	out.Example(fmt.Sprintf("\ndupers %s <bucket name>", cmd))
	return 1
}

// Export the bucket as a CSV file.
func Export(db *bolt.DB, quiet bool, args [2]string) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	const x = "export"
	if code := Check(x, x, args[1]); code > 0 {
		return ErrBucketNil
	}
	name, err := database.Abs(args[1])
	if err != nil {
		return err
	}
	w := os.Stdout
	if err := database.Exist(db, name); err != nil {
		if errors.Is(err, bolt.ErrBucketNotFound) { // TODO: move this out to the main.go
			fmt.Fprintf(w, "Bucket name: %s\n", name)
			out.Example("\ndupers export <bucket name>")
		}
		return err
	}
	exp, err := database.CSVExport(db, name)
	if err != nil {
		return err
	}
	s := fmt.Sprintf("%s %s\n", color.Secondary.Sprint("Bucket name:"), color.Debug.Sprint(name))
	s += fmt.Sprintf("The exported bucket file is at: %s", exp)
	out.Response(s, quiet)
	if quiet {
		fmt.Fprintln(os.Stdout, exp)
	}
	return nil
}

// Import a CSV file into the database.
func Import(db *bolt.DB, quiet, assumeYes bool, args [2]string) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	if args[1] == "" {
		fmt.Fprintln(os.Stderr, "Cannot import file as no filepath was provided.")
		out.Example(fmt.Sprintf("\ndupers %s <filepath>", dim))
		return ErrImport
	}
	name, err := database.Abs(args[1])
	if err != nil {
		return err
	}
	r, err := database.CSVImport(db, name, assumeYes)
	if err != nil {
		return err
	}
	p := message.NewPrinter(language.English)
	s := p.Sprintf("\rSuccessfully imported %d records.", number.Decimal(r))
	out.Response(s, quiet)
	return nil
}

// List the content of a bucket to the stdout.
func List(db *bolt.DB, quiet bool, args [2]string) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	if code := Check("list", dls, args[1]); code > 0 {
		return ErrBucketNil
	}
	name, err := database.Abs(args[1])
	if err != nil {
		return err
	}
	ls, err := database.List(db, name)
	if err != nil {
		return err
	}
	// sort the filenames
	names := make([]string, 0, len(ls))
	for name := range ls {
		names = append(names, string(name))
	}
	w := os.Stdout
	sort.Strings(names)
	for _, name := range names {
		sum := ls[database.Filepath(name)]
		fmt.Fprintf(w, "%x %s\n", sum, name)
	}
	if cnt := len(ls); !quiet && cnt > 0 {
		p := message.NewPrinter(language.English)
		fmt.Fprintf(w, "%s %s\n", color.Primary.Sprint(p.Sprint(number.Decimal(cnt))),
			color.Secondary.Sprint("items listed. Checksums are 32 byte, SHA-256 (FIPS 180-4)."))
	}
	return nil
}

// Move renames a bucket by duplicating it to a new bucket location.
func Move(db *bolt.DB, c *dupe.Config, assumeYes bool, args [2]string) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	if c == nil {
		return dupe.ErrNilConfig
	}
	b, dir := args[0], args[1]
	if code := Check("move and rename", dmv, b); code > 0 {
		return ErrBucketNil
	}
	name, err := database.Abs(b)
	if err != nil {
		return err
	}
	w := os.Stdout
	if err := database.Exist(db, name); err != nil {
		if errors.Is(err, bolt.ErrBucketNotFound) {
			fmt.Fprintf(w, "\nBucket name: %s\n", name)
			out.Example("dupers mv <bucket name> <new directory>")
		}
		return fmt.Errorf("%w: %s", err, name)
	}
	if dir == "" {
		fmt.Fprintln(os.Stderr, "\nCannot move bucket within the database as no new directory was provided.")
		out.Example(fmt.Sprintf("dupers mv %s <new directory>", b))
		return ErrNewName
	}
	newName, err := database.Abs(dir)
	if err != nil {
		return err
	}
	if !c.Quiet {
		fmt.Fprintf(w, "%s\t%s\n%s\t%s\n",
			color.Secondary.Sprint("Bucket name:"), color.Debug.Sprint(name),
			"New name:", color.Debug.Sprint(newName))
		fmt.Fprintln(w, "Renames the database bucket, but this does not make changes to the file system.")
		if !out.AskYN("Rename bucket", assumeYes, out.No) {
			return nil
		}
	}
	if err := database.Rename(db, name, newName); err != nil {
		if errors.Is(err, database.ErrSameNames) {
			fmt.Fprintln(os.Stderr, "\nCannot move the bucket to the same directory as its current directory.")
			out.Example(fmt.Sprintf("dupers mv %s <new directory>\n", b))
		}
		return err
	}
	return nil
}

// Remove the bucket from the database.
func Remove(db *bolt.DB, quiet, assumeYes bool, args [2]string) error {
	if db == nil {
		return bolt.ErrBucketNotFound
	}
	bucket := args[1]
	if code := Check("remove", drm, bucket); code > 0 {
		return ErrBucketNil
	}
	name, err := database.Abs(bucket)
	if err != nil {
		return err
	}
	items, err := database.Count(db, name)
	if err != nil {
		return err
	}
	if !quiet {
		w := os.Stdout
		fmt.Fprintf(w, "%s\t%s\n", color.Secondary.Sprint("Bucket:"), color.Debug.Sprint(name))
		p := message.NewPrinter(language.English)
		fmt.Fprintf(w, "%s\t%s\n", color.Secondary.Sprint("Items:"), color.Debug.Sprint(p.Sprint(items)))
		if !out.AskYN("Remove this bucket", assumeYes, out.No) {
			return nil
		}
	}
	if err := rmBucket(db, name, bucket); err != nil {
		return err
	}
	s := fmt.Sprintf("Removed bucket from the database: '%s'\n", name)
	out.Response(s, quiet)
	return nil
}

func rmBucket(db *bolt.DB, name, retry string) error {
	err := database.RM(db, name)
	if errors.Is(err, bolt.ErrBucketNotFound) {
		// retry with the original argument
		if err1 := database.RM(db, retry); err1 != nil {
			if errors.Is(err1, bolt.ErrBucketNotFound) {
				return notFound(db, name, err1)
			}
			return err
		}
	}
	return err
}

func notFound(db *bolt.DB, name string, err error) error {
	if db == nil {
		return bolt.ErrBucketNotFound
	}
	out.StderrCR(err)
	w := os.Stdout
	fmt.Fprintf(w, "Bucket to remove: %s\n", color.Danger.Sprint(name))
	buckets, err2 := database.All(db)
	if err2 != nil {
		return err2
	}
	if len(buckets) == 0 {
		fmt.Fprintln(w, "There are no buckets in the database")
		out.ErrFatal(nil)
	}
	fmt.Fprintf(w, "Buckets in use:   %s\n",
		strings.Join(buckets, "\n\t\t  "))
	out.ErrFatal(nil)
	return nil
}

// Rescan the bucket for changes with the file system.
func Rescan(db *bolt.DB, c *dupe.Config, archives bool, args [2]string) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	cmd := dup
	if archives {
		cmd = dupp
	}
	if code := Check("add or update", cmd, args[1]); code > 0 {
		return ErrBucketNil
	}
	path, err := database.Abs(args[1])
	if err != nil {
		return err
	}
	name := dupe.Bucket(path)
	if archives {
		if err := c.WalkArchiver(db, name); err != nil {
			return err
		}
	} else if err := c.WalkDir(db, name); err != nil {
		return err
	}
	if !c.Quiet {
		fmt.Fprintln(os.Stdout, c.Status())
	}
	return nil
}
