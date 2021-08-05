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

	bolt "go.etcd.io/bbolt"
)

const (
	PrivateFile fs.FileMode = 0600
	PrivateDir  fs.FileMode = 0700

	Test1    = "../test/bucket1"
	Test2    = "../test/bucket2"
	SevenZip = "../test/randomfiles.7z"
	Source1  = Test1 + "/0vlLaUEvzAWP"
	dbName   = "dupers.db"
	dbPath   = "dupers"
	oneKb    = 1024
	oneMb    = oneKb * oneKb
)

var (
	ErrBucket = errors.New("bucket already exists")
	ErrCreate = errors.New("create bucket")
)

// Bucket1 returns the absolute path of bucket test 1.
func Bucket1() string {
	b, err := filepath.Abs(Test1)
	if err != nil {
		log.Fatal(err)
	}
	return b
}

// Bucket2 returns the absolute path of bucket test 2.
func Bucket2() string {
	b, err := filepath.Abs(Test2)
	if err != nil {
		log.Fatal(err)
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

// Item1 returns the absolute path of test source file 1.
func Item1() string {
	b, err := filepath.Abs(Source1)
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
	fmt.Println(path)
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
func TestOpen() error {
	path, err := Name()
	if err != nil {
		return err
	}
	fmt.Println(path)
	db, err := bolt.Open(path, PrivateFile, nil)
	if err != nil {
		return err
	}
	defer db.Close()
	return db.Update(func(tx *bolt.Tx) error {
		fmt.Println(db.Path())
		b, err := tx.CreateBucketIfNotExists([]byte(Bucket1()))
		if err != nil {
			return err
		}
		sum256, err := Read(Item1())
		if err != nil {
			return err
		}
		return b.Put([]byte(Item1()), sum256[:])
	})
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
	if err != nil {
		return err
	}
	return nil
}
