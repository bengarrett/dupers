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
	DirMode  fs.FileMode = 0700
	FileMode fs.FileMode = 0600

	SevenZip = "../test/randomfiles.7z"
	Source1  = Test1 + "/0vlLaUEvzAWP"
	Test1    = "../test/bucket1"
	Test2    = "../test/bucket2"
	dbName   = "dupers.db"
	dbPath   = "dupers"
	oneKb    = 1024
	oneMb    = oneKb * oneKb
)

var (
	ErrBucket = errors.New("bucket already exists")
	ErrCreate = errors.New("create bucket")
)

func Bucket1() string {
	b, err := filepath.Abs(Test1)
	if err != nil {
		log.Fatal(err)
	}
	return b
}

func Item1() string {
	b, err := filepath.Abs(Source1)
	if err != nil {
		log.Fatal(err)
	}
	return b
}

func Source2() string {
	b, err := filepath.Abs(Test2)
	if err != nil {
		log.Fatal(err)
	}
	return b
}

func CreateItem(bucket, item string, db *bolt.DB) error {
	if db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	return db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucket([]byte(bucket))
		if err != nil {
			return err
		}
		sum256, err := Read(item)
		if err != nil {
			return err
		}
		return b.Put([]byte(item), sum256[:])
	})
}

// Name returns the absolute path of the Bolt database.
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
		if err1 := os.MkdirAll(dir, DirMode); err != nil {
			return "", err1
		}
	}
	return filepath.Join(dir, dbName), nil
}

func DBForConfig() (*bolt.DB, error) {
	path, err := Name()
	if err != nil {
		return nil, err
	}
	fmt.Println(path)
	db, err := bolt.Open(path, FileMode, nil)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func DBUp() error {
	path, err := Name()
	if err != nil {
		return err
	}
	fmt.Println(path)
	db, err := bolt.Open(path, FileMode, nil)
	if err != nil {
		return err
	}
	defer db.Close()
	return db.Update(func(tx *bolt.Tx) error {
		fmt.Println(db.Path())
		b, err := tx.CreateBucket([]byte(Bucket1()))
		if err != nil {
			if errors.As(err, &ErrBucket) {
				return nil
			}
			return fmt.Errorf("%w: %s", ErrCreate, err)
		}
		sum256, err := Read(Item1())
		if err != nil {
			return err
		}
		return b.Put([]byte(Item1()), sum256[:])
	})
}

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

func DBDown() error {
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
