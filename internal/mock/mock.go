// © Ben Garrett https://github.com/bengarrett/dupers

// Mock is a set of simulated database and bucket functions for unit testing.
package mock

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	bolt "go.etcd.io/bbolt"
)

const (
	PrivateFile fs.FileMode = 0o600
	PrivateDir  fs.FileMode = 0o700

	csv1     = "test/export-bucket1.csv"
	test1    = "test/bucket1"
	test2    = "test/bucket2"
	SevenZip = "test/randomfiles.7z"
	source1  = "/0vlLaUEvzAWP"
	dbName   = "dupers.db"
	dbPath   = "dupers"
	win      = "windows"
	oneKb    = 1024
	oneMb    = oneKb * oneKb
)

var (
	ErrBucket = errors.New("bucket already exists")
	ErrCreate = errors.New("create bucket")
)

func RootDir() string {
	_, b, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	return filepath.Join(filepath.Dir(b), "../..")
}

// Bucket1 returns the absolute path of bucket test 1.
func Bucket1() string {
	path := filepath.Join(RootDir(), test1)
	b, err := filepath.Abs(path)
	if err != nil {
		log.Fatal(err)
	}

	if runtime.GOOS == win {
		b = strings.ToLower(b)
	}

	return b
}

// Bucket2 returns the absolute path of bucket test 2.
func Bucket2() string {
	path := filepath.Join(RootDir(), test2)
	b, err := filepath.Abs(path)
	if err != nil {
		log.Fatal(err)
	}

	if runtime.GOOS == win {
		b = strings.ToLower(b)
	}

	return b
}

// CreateItem adds the bucket and the named file to the database.
func CreateItem(bucket, file string, db *bolt.DB) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}

	return db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}
		sum256, err := Read(file)
		if err != nil {
			return err
		}
		return b.Put([]byte(file), sum256[:])
	})
}

// Export1 returns the absolute path of export csv file 1.
func Export1() string {
	path := filepath.Join(RootDir(), csv1)
	f, err := filepath.Abs(path)
	if err != nil {
		log.Fatal(err)
	}

	return f
}

// Item1 returns the absolute path of test source file 1.
func Item1() string {
	path := filepath.Join(Bucket1(), source1)
	b, err := filepath.Abs(path)
	if err != nil {
		log.Fatal(err)
	}

	return b
}

// Open and return the mock database.
func Open() (*bolt.DB, error) {
	path, err := Name()
	if err != nil {
		return nil, err
	}

	fmt.Fprintln(os.Stdout, "open mock db:", path)
	db, err := bolt.Open(path, PrivateFile, nil)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// Name returns the absolute path of the mock Bolt database.
func Name() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir, err = os.UserHomeDir()

		if err != nil {
			return "", err
		}
	}

	dir = filepath.Join(dir, dbPath, "test")

	_, err = os.Stat(dir)
	if os.IsNotExist(err) {
		if err1 := os.MkdirAll(dir, PrivateDir); err != nil {
			return "", err1
		}
	}

	return filepath.Join(dir, dbName), nil
}

// Read the named file and return its SHA256 checksum.
func Read(name string) (sum [32]byte, err error) {
	f, err := os.Open(name)
	if err != nil {
		return [32]byte{}, err
	}
	defer f.Close()

	buf, h := make([]byte, oneMb), sha256.New()
	if _, err := io.CopyBuffer(h, f, buf); err != nil {
		return [32]byte{}, err
	}

	copy(sum[:], h.Sum(nil))

	return sum, nil
}

// TestOpen creates and opens the mock database, the test 1 bucket and adds the source 1 file.
// The mock database is closed after the update.
// Note: If this test fails under Windows, try running `go test ./...` after closing VS Code.
// https://github.com/electron-userland/electron-builder/issues/3666
func TestOpen() error {
	path, err := Name()
	if err != nil {
		return err
	}

	db, err := bolt.Open(path, PrivateFile, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.Update(func(tx *bolt.Tx) error {
		// delete any existing buckets from the mock database
		if err := tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			return tx.DeleteBucket(name)
		}); err != nil {
			return err
		}
		// create the new mock bucket
		b, err := tx.CreateBucket([]byte(Bucket1()))
		if err != nil {
			return err
		}
		sum256, err := Read(Item1())
		if err != nil {
			return err
		}
		return b.Put([]byte(Item1()), sum256[:])
	}); err != nil {
		return err
	}

	return db.Close()
}

// TestRemove deletes the mock database.
func TestRemove() error {
	path, err := Name()
	if err != nil {
		return err
	}

	err = os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if runtime.GOOS == "windows" {
		var e *os.PathError
		if errors.As(err, &e) {
			log.Printf("could not remove the mock database as the Windows filesystem has locked it: %s\n", path)
			return nil
		}
	}

	if err != nil {
		return err
	}

	return nil
}
