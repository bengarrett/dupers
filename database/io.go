// © Ben Garrett https://github.com/bengarrett/dupers

// Package database interacts with Dupers bbolt database and buckets.
package database

import (
	"bufio"
	csvEnc "encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bengarrett/dupers/database/internal/csv"
	"github.com/bengarrett/dupers/internal/out"
	"github.com/gookit/color"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

const (
	whichBkt = "What bucket name do you wish to use"
	loops    = 0
)

var (
	ErrBucketExists = errors.New("bucket already exists in the database")
	ErrBucketNotDir = errors.New("bucket path is not a directory")
	ErrBucketPath   = errors.New("directory used by the bucket does not exist on your system")
	ErrImportList   = errors.New("import list is empty")
)

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
			if bucket = csv.BucketName(line); bucket == "" {
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

// read bolt option to open in read only mode with a file lock timeout.
func read() *bolt.Options {
	return &bolt.Options{ReadOnly: true, Timeout: Timeout}
}

// write bolt option to open in write mode with a file lock timeout.
func write() *bolt.Options {
	return &bolt.Options{Timeout: Timeout}
}

// OpenRead opens the Bolt database for reading.
func OpenRead() (db *bolt.DB, err error) {
	path, err := DB()
	if err != nil {
		return nil, err
	}
	db, err = bolt.Open(path, PrivateFile, read())
	if err != nil {
		return nil, err
	}
	return db, nil
}

// OpenRead opens the Bolt database for writing and reading.
func OpenWrite() (db *bolt.DB, err error) {
	path, err := DB()
	if err != nil {
		return nil, err
	}
	db, err = bolt.Open(path, PrivateFile, write())
	if err != nil {
		return nil, err
	}
	return db, nil
}

// Backup makes a copy of the database to the named location.
func Backup() (name string, written int64, err error) {
	src, err := DB()
	if err != nil {
		return "", 0, err
	}
	dir, err := Home()
	if err != nil {
		return "", 0, err
	}
	name = filepath.Join(dir, backup())
	written, err = CopyFile(src, name)
	if err != nil {
		return "", 0, err
	}
	return name, written, nil
}

// backupName generates a time sensitive filename for the backup.
func backup() string {
	now, ext := time.Now().Format(backupTime), filepath.Ext(boltName)
	return fmt.Sprintf("%s-backup-%s%s", strings.TrimSuffix(boltName, ext), now, ext)
}

// CopyFile duplicates the named file to the destination filepath.
func CopyFile(name, dest string) (int64, error) {
	// read source
	f, err := os.Open(name)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	// create backup file
	bu, err := os.Create(dest)
	if err != nil {
		return 0, err
	}
	defer bu.Close()
	// duplicate data
	return io.Copy(bu, f)
}

// ExportCSV saves the bucket data to an export csv file.
// The generated file is RFC 4180 compatible using comma-separated values.
func ExportCSV(bucket string, db *bolt.DB) (name string, err error) {
	if db == nil {
		db, err = OpenRead()
		if err != nil {
			return "", err
		}
		defer db.Close()
	}

	dir, err := Home()
	if err != nil {
		return "", err
	}
	name = filepath.Join(dir, export())
	f, err := os.Create(name)
	if err != nil {
		return "", err
	}
	defer f.Close()

	meta := strings.Join([]string{"path", bucket}, "#")
	r := [][]string{
		{"sha256_sum", meta},
	}
	ls, errLS := List(bucket, db)
	if errLS != nil {
		return "", err
	}
	for file, sum := range ls {
		rel := strings.TrimPrefix(string(file), bucket)
		r = append(r, []string{fmt.Sprintf("%x", sum), rel})
	}
	w := csvEnc.NewWriter(f)
	if err := w.WriteAll(r); err != nil {
		return "", err
	}
	return name, nil
}

// exportName generates a time sensitive filename for the export.
func export() string {
	now, ext := time.Now().Format(backupTime), filepath.Ext(csvName)
	return fmt.Sprintf("%s-%s%s", strings.TrimSuffix(csvName, ext), now, ext)
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

// ImportCSV reads the named export csv file and imports its content to the database.
func ImportCSV(name string, db *bolt.DB) (records int, err error) {
	if db == nil {
		db, err = OpenWrite()
		if err != nil {
			return 0, err
		}
		defer db.Close()
	}

	file, err := os.Open(name)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	if err1 := csv.Checker(file); err1 != nil {
		return 0, err1
	}
	bucket, lists, err2 := Scanner(file)
	if err2 != nil {
		return 0, err2
	}
	bucket, err = bucketChk(bucket, db)
	if err != nil {
		return 0, err
	}
	items := 0
	for range *lists {
		items++
	}
	p := message.NewPrinter(language.English)
	s := "\n"
	s += color.Secondary.Sprint("Found ") +
		color.Primary.Sprintf("%s valid items", p.Sprint(number.Decimal(items))) +
		color.Secondary.Sprint(" in the CSV file.")
	fmt.Println(s)
	s = color.Secondary.Sprint("These will be added to the bucket: ")
	s += color.Debug.Sprint(bucket)
	fmt.Println(s)
	return Import(Bucket(bucket), lists, db)
}

// bucketChk checks the validity and usage of the named bucket in the database.
func bucketChk(name string, db *bolt.DB) (bucket string, err error) {
	if db == nil {
		db, err = OpenRead()
		if err != nil {
			return "", err
		}
		defer db.Close()
	}

	for {
		if err := db.View(func(tx *bolt.Tx) error {
			if b := tx.Bucket([]byte(name)); b == nil {
				bucket = name
				if stat(bucket) {
					return nil
				}
			}
			if bucket = bucketRename(name); bucket != "" {
				if stat(bucket) {
					return nil
				}
			}
			return nil
		}); err != nil {
			return "", err
		}
		if bucket != "" {
			return bucket, nil
		}
	}
}

func stat(bucket string) bool {
	for {
		fmt.Println()
		if bucket = bucketStat(bucket); bucket != "" {
			return true
		}
	}
}

// bucketRename prompts for confirmation for the use of the named bucket.
func bucketRename(name string) string {
	out.ErrCont(ErrBucketExists)
	fmt.Printf("\nImport bucket name: %s\n\n", color.Debug.Sprint(name))
	fmt.Println("The existing data in this bucket will overridden and any new data will be appended.")
	if out.YN("Do you want to continue using this bucket", out.Yes) {
		return name
	}
	fmt.Println("\nPlease choose a new bucket, which must be an absolute directory path.")
	return out.Prompt(whichBkt)
}

// bucketStat checks the validity of the named bucket and prompts for user confirmation on errors.
func bucketStat(name string) string {
	printName := func() {
		fmt.Printf("\nImport bucket directory: %s\n\n", color.Debug.Sprint(name))
	}

	for {
		name = strings.TrimSpace(name)
		abs, err := Abs(name)
		if err != nil {
			out.ErrCont(fmt.Errorf("%w: %s", err, name))

			continue
		}
		s, err := os.Stat(abs)
		if errors.Is(err, os.ErrNotExist) {
			out.ErrCont(ErrBucketPath)
			printName()
			fmt.Println("You may still run dupe checks and searches without the actual files on your system.")
			fmt.Println("Choosing no will prompt for a new bucket.")
			if out.YN("Do you want to continue using this bucket", out.Yes) {
				return abs
			}
			name = out.Prompt(whichBkt)

			continue
		} else if err == nil && !s.IsDir() {
			err = ErrBucketNotDir
		}
		if err != nil {
			out.ErrCont(ErrBucketNotDir)
			printName()
			fmt.Println("You cannot use this path as a bucket, please choose an absolute directory path.")
			name = out.Prompt(whichBkt)

			continue
		}
		return abs
	}
}

// Import the list of data and save it to the database.
// If the named bucket does not exist, it is created.
func Import(name Bucket, ls *Lists, db *bolt.DB) (imported int, err error) {
	if ls == nil {
		return 0, ErrImportList
	}
	if db == nil {
		var err error
		db, err = OpenWrite()
		if err != nil {
			return 0, err
		}
		defer db.Close()
	}
	const batchItems = 50000
	items, total := 0, len(*ls)
	batch := make(Lists, batchItems)
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
	for path, sum := range batch {
		if err := db.Update(func(tx *bolt.Tx) error {
			b, err := tx.CreateBucketIfNotExists([]byte(name))
			if err != nil {
				return err
			}
			fmt.Print(out.Status(imported, total, out.Read))
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
