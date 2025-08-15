// © Ben Garrett https://github.com/bengarrett/dupers
package database

import (
	"bufio"
	csvEnc "encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bengarrett/dupers/internal/printer"
	"github.com/bengarrett/dupers/pkg/database/bucket"
	"github.com/bengarrett/dupers/pkg/database/csv"
	"github.com/gookit/color"
	bolt "go.etcd.io/bbolt"
	bberr "go.etcd.io/bbolt/errors"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

const (
	Timeout = 3 * time.Second // Timeout lock option for the Bolt database.
	loops   = 0
)

var (
	ErrImportList = errors.New("import list cannot be empty")
	ErrNoBucket   = errors.New("the named bucket cannot be empty")
	ErrNoDest     = errors.New("the destination path cannot be empty")
	ErrNoFilename = errors.New("the named file cannot be empty")
)

// Backup makes a copy of the database to the named location.
//
// Returned is the path to the database and the number of bytes copied.
func Backup() (string, int64, error) {
	src, err := DB()
	if err != nil {
		return "", 0, err
	}
	dir, err := Home()
	if err != nil {
		return "", 0, err
	}
	name := filepath.Join(dir, backup())
	written, err := CopyFile(src, name)
	if err != nil {
		return "", 0, err
	}
	return name, written, nil
}

// backup generates a time sensitive name for the backup file.
func backup() string {
	now, ext := time.Now().Format(backupTime), filepath.Ext(boltName)
	return fmt.Sprintf("%s-backup-%s%s", strings.TrimSuffix(boltName, ext), now, ext)
}

// CopyFile duplicates the named file to the destination filepath.
func CopyFile(name, dest string) (int64, error) {
	if name == "" {
		return 0, ErrNoFilename
	}
	if dest == "" {
		return 0, ErrNoDest
	}
	// read source
	f, err := os.Open(name)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	// create backup file
	bf, err := os.Create(dest)
	if err != nil {
		return 0, err
	}
	defer bf.Close()
	// duplicate data
	return io.Copy(bf, f)
}

// CSVExport saves the bucket data to an export csv file.
// The generated file is RFC 4180 compatible using comma-separated values.
func CSVExport(db *bolt.DB, bucket string) (string, error) {
	if db == nil {
		return "", bberr.ErrDatabaseNotOpen
	}
	dir, err := Home()
	if err != nil {
		return "", err
	}
	name := filepath.Join(dir, export())
	dest, err := os.Create(name)
	if err != nil {
		return "", err
	}
	defer dest.Close()

	meta := strings.Join([]string{"path", bucket}, "#")
	records := [][]string{
		{"sha256_sum", meta},
	}
	ls, errLS := List(db, bucket)
	if errLS != nil {
		return "", errLS
	}
	for file, sum := range ls {
		rel := strings.TrimPrefix(string(file), bucket)
		records = append(records, []string{hex.EncodeToString(sum[:]), rel})
	}
	w := csvEnc.NewWriter(dest)
	if err := w.WriteAll(records); err != nil {
		return "", err
	}
	return name, nil
}

// export generates a time sensitive name for the export file.
func export() string {
	now, ext := time.Now().Format(backupTime), filepath.Ext(csvName)
	return fmt.Sprintf("%s-%s%s", strings.TrimSuffix(csvName, ext), now, ext)
}

// CSVImport reads the named csv export file and imports its content to the database.
func CSVImport(db *bolt.DB, name string, assumeYes bool) (int, error) {
	if db == nil {
		return 0, bberr.ErrDatabaseNotOpen
	}

	file, err := os.Open(name)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	if err := csv.Checker(file); err != nil {
		return 0, err
	}
	name, lists, err := Scanner(file)
	if err != nil {
		return 0, err
	}
	items := 0
	for range *lists {
		items++
	}
	w := os.Stdout
	p := message.NewPrinter(language.English)
	s := "\n"
	s += color.Secondary.Sprint("Found ") +
		color.Primary.Sprintf("%s valid items", p.Sprint(number.Decimal(items))) +
		color.Secondary.Sprint(" in the CSV file.")
	fmt.Fprintln(w, s)
	s = color.Secondary.Sprint("These will be added to the bucket: ")
	s += color.Debug.Sprint(name)
	fmt.Fprintln(w, s)
	return Import(db, Bucket(name), lists)
}

// Home returns the user's home directory.
// Or if that fails, returns the current working directory.
func Home() (string, error) {
	s, err := os.UserHomeDir()
	if err != nil {
		if s, err = os.Getwd(); err != nil {
			return "", err
		}
	}
	return s, err
}

// Import the list of data and save it to the database.
// If the named bucket does not exist, it is created.
//
// The returned int is the number of records imported.
func Import(db *bolt.DB, name Bucket, ls *Lists) (int, error) {
	if db == nil {
		return 0, bberr.ErrDatabaseNotOpen
	}
	if ls == nil {
		return 0, ErrImportList
	}
	const batchItems = 50000
	imported := 0
	items, total := 0, len(*ls)
	batch := make(Lists, batchItems)
	var err error
	for path, sum := range *ls {
		batch[path] = sum
		items++
		if items%batchItems == 0 {
			imported, err = batch.iterate(db, name, imported, total)
			if err != nil {
				return 0, err
			}
			batch = make(Lists, batchItems)
			continue
		}
	}
	if len(batch) > 0 {
		imported, err = batch.iterate(db, name, imported, total)
		if err != nil {
			return 0, err
		}
	}
	return imported, nil
}

func (batch Lists) iterate(db *bolt.DB, name Bucket, imported, total int) (int, error) {
	if db == nil {
		return 0, bberr.ErrDatabaseNotOpen
	}
	for path, sum := range batch {
		if err := db.Update(func(tx *bolt.Tx) error {
			b, err := tx.CreateBucketIfNotExists([]byte(name))
			if err != nil {
				return err
			}
			fmt.Fprint(os.Stdout, printer.Status(imported, total, printer.Read))
			if err := b.Put([]byte(string(path)), sum[:]); err != nil {
				return err
			}
			imported++
			return nil
		}); err != nil {
			return 0, err
		}
	}
	return imported, nil
}

// OpenRead opens the Bolt database for reading.
func OpenRead() (*bolt.DB, error) {
	path, err := DB()
	if err != nil {
		return nil, err
	}
	db, err := bolt.Open(path, PrivateFile, read())
	if err != nil {
		return nil, err
	}
	return db, nil
}

// read bolt option to open in read only mode with a file lock timeout.
func read() *bolt.Options {
	return &bolt.Options{ReadOnly: true, Timeout: Timeout}
}

// OpenRead opens the Bolt database for writing and reading.
func OpenWrite() (*bolt.DB, error) {
	path, err := DB()
	if err != nil {
		return nil, err
	}
	db, err := bolt.Open(path, PrivateFile, write())
	if err != nil {
		return nil, err
	}
	return db, nil
}

// write bolt option to open in write mode with a file lock timeout.
func write() *bolt.Options {
	return &bolt.Options{Timeout: Timeout}
}

// Scanner reads the content of an export csv file.
// It returns the stored bucket and csv data as a List ready for import.
func Scanner(file *os.File) (string, *Lists, error) {
	if file == nil {
		return "", nil, csv.ErrFileNoDesc
	}
	const firstItem = 2
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	bucket, row := "", 0
	lists := make(Lists)

	for scanner.Scan() {
		row++
		line := scanner.Text()
		if row == 1 {
			if bucket = csv.Bucket(line); bucket == "" {
				return "", nil, fmt.Errorf("%w, invalid header: %s", csv.ErrImportFile, line)
			}

			continue
		}
		sum, key, err := csv.Import(line, bucket)
		if err != nil {
			if row == firstItem {
				return "", nil, err
			}

			continue
		}
		if loops != 0 && row > loops+1 {
			break
		}
		lists[Filepath(key)] = sum
	}
	return bucket, &lists, nil
}

// Usage checks the validity and usage of the named bucket in the database.
func Usage(db *bolt.DB, name string, assumeYes bool) (string, error) {
	if db == nil {
		return "", bberr.ErrDatabaseNotOpen
	}
	if name == "" {
		return "", ErrNoBucket
	}
	for {
		path := ""
		if err := db.View(func(tx *bolt.Tx) error {
			if b := tx.Bucket([]byte(name)); b == nil {
				path = name
				if bucket.Stats(path, assumeYes) {
					return nil
				}
			}
			if path = bucket.Rename(name, assumeYes); path != "" {
				if bucket.Stats(path, assumeYes) {
					return nil
				}
			}
			return nil
		}); err != nil {
			return "", err
		}
		if path != "" {
			return path, nil
		}
	}
}
