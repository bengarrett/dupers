// © Ben Garrett https://github.com/bengarrett/dupers

// Mock is a set of simulated database and bucket functions for unit testing.
package mock

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/bengarrett/dupers/pkg/database"
	bolt "go.etcd.io/bbolt"
)

const (
	PrivateFile fs.FileMode = 0o600                 // PrivateFile mode means only the owner has read/write access.
	PrivateDir  fs.FileMode = 0o700                 // PrivateDir mode means only the owner has read/write/dir access.
	SevenZip                = "test/randomfiles.7z" //
	NoSuchFile              = "qwertryuiop"         // NoSuchFile is a non-existent filename.
	filename                = "dupers.db"           // filename of the mock database.
	subdir                  = "dupers-mock"         // subdir is the sub-directory within config that houses the mock database.
	win                     = "windows"
	oneKb                   = 1024
	oneMb                   = oneKb * oneKb
)

var (
	ErrBucket    = errors.New("mock bucket number does not exist")
	ErrExport    = errors.New("mock export number does not exist")
	ErrExtension = errors.New("mock file for extension does not exist")
	ErrItem      = errors.New("mock item number does not exist")
	ErrLockedDB  = errors.New("mock database is locked by the Windows filesystem")
	ErrNoRoot    = errors.New("could not determine the root directory")
)

var sources = map[int]string{
	0: "/0vlLaUEvzAWP",
	1: "/3a9dnxgSVEnJ",
	2: "/12wZkDDR9CQ0",
}

var checksums = map[int]string{
	0: "1a1d76a3187ccee147e6c807277273afbad5d2680f5eadf1012310743e148f22",
	1: "1bdd103eace1a58d2429d447ac551030a9da424056d2d89a77b1366a04f1f1cc",
	2: "c5f338d4057fb107793032de91b264707c3c27bf9970687a78a080a4bf095c26",
}

var extensions = map[string]string{
	"7z":  "/randomfiles.7z",
	"xz":  "/randomfiles.tar.xz",
	"txt": "/randomfiles.txt",
	"zip": "/randomfiles.zip",
}

// RootDir returns the root directory of the program's source code.
// An empty string is returned if the directory cannot be determined.
func RootDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	return filepath.Join(filepath.Dir(file), "..", "..")
}

// TempDir returns a hidden tmp mock directory path within the
// root directory of the program's source code.
// If the directory doesn't exist, it is created.
func TempDir() (string, error) {
	root := RootDir()
	if root == "" {
		return "", ErrNoRoot
	}
	tmp := filepath.Join(root, ".tmp", "mock")
	if err := os.MkdirAll(tmp, PrivateDir); err != nil {
		return tmp, err
	}
	return tmp, nil
}

// Bucket returns the absolute path of test bucket.
func Bucket(i int) (string, error) {
	name := ""
	switch i {
	case 1:
		name = "bucket1"
	case 2:
		name = "bucket2"
	default:
		return "", ErrBucket
	}
	path := filepath.Join(RootDir(), "test", name)
	f, err := filepath.Abs(path)
	if err != nil {
		log.Fatal(err)
	}

	if runtime.GOOS == win {
		f = strings.ToLower(f)
	}

	return f, nil
}

// Export returns the absolute path of export csv file for a bucket.
func Export(i int) (string, error) {
	if i >= len(sources) || i < 0 {
		return "", ErrItem
	}
	filename := fmt.Sprintf("export-bucket%d.csv", i)
	path := filepath.Join(RootDir(), "test", filename)
	f, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	return f, nil
}

// Item returns the absolute path of test source file item.
func Item(i int) (string, error) {
	if i >= len(sources) || i < 0 {
		return "", ErrItem
	}
	elem := sources[i]
	bucket1, err := Bucket(1)
	if err != nil {
		return "", err
	}
	path := filepath.Join(bucket1, elem)
	f, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	return f, nil
}

// Extension returns the absolute path of a test file based on an extension.
func Extension(ext string) (string, error) {
	elem, ok := extensions[ext]
	if !ok {
		return "", ErrExtension
	}
	path := filepath.Join(RootDir(), "test", elem)
	f, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	return f, nil
}

// NamedDB returns the absolute path of the mock Bolt database.
func NamedDB() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir, err = os.UserHomeDir()

		if err != nil {
			return "", err
		}
	}

	path := filepath.Join(dir, subdir)

	if _, err = os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(path, PrivateDir); err != nil {
				return "", fmt.Errorf("%w: %s", err, path)
			}
		}
	} else if err != nil {
		return "", err
	}

	return filepath.Join(path, filename), nil
}

// Create the mock database and return its location.
// Note: If this test fails under Windows, try running `go test ./...` after closing VS Code.
// https://github.com/electron-userland/electron-builder/issues/3666
func Create() (string, error) {
	path, err := NamedDB()
	if err != nil {
		return "", err
	}
	db, err := bolt.Open(path, PrivateFile, nil)
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, path)
	}
	defer db.Close()
	err = db.Update(func(tx *bolt.Tx) error {
		// delete any existing buckets from the mock database
		if err := tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			return tx.DeleteBucket(name)
		}); err != nil {
			return err
		}
		// create the new mock bucket #1
		bucket1, err := Bucket(1)
		if err != nil {
			return err
		}
		b, err := tx.CreateBucket([]byte(bucket1))
		if err != nil {
			return fmt.Errorf("%w: create bucket: %s", err, bucket1)
		}
		// create the new, but empty mock bucket #2
		bucket2, err := Bucket(2)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucket([]byte(bucket2))
		if err != nil {
			return fmt.Errorf("%w: create bucket: %s", err, bucket1)
		}
		for i := range sources {
			item, err := Item(i)
			if err != nil {
				return fmt.Errorf("%w: get item %d", err, i)
			}
			sum256, err := Read(item)
			if err != nil {
				return fmt.Errorf("%w: read item %d: %s", err, i, item)
			}
			if err := b.Put([]byte(item), sum256[:]); err != nil {
				return fmt.Errorf("%w: put item %d", err, i)
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return path, nil
}

// Read the named file and return its SHA256 checksum.
func Read(name string) (sum [32]byte, err error) {
	f, err := os.Open(name)
	if err != nil {
		return [32]byte{}, err
	}
	defer f.Close()

	buf := make([]byte, oneMb)
	h := sha256.New()
	if _, err := io.CopyBuffer(h, f, buf); err != nil {
		return [32]byte{}, err
	}

	// copy(sum[:], h.Sum(nil))

	// x := [32]byte(h.Sum(nil))

	return [32]byte(h.Sum(nil)), nil
}

// Database creates, opens and returns the mock database.
func Database() (*bolt.DB, error) {
	if err := Delete(); err != nil {
		return nil, err
	}
	if _, err := Create(); err != nil {
		return nil, err
	}
	return Open()
}

// Open the mock database.
// This will need to be closed after use.
func Open() (*bolt.DB, error) {
	path, err := NamedDB()
	if err != nil {
		return nil, err
	}

	db, err := bolt.Open(path, PrivateFile, nil)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// Delete the mock database.
func Delete() error {
	path, err := NamedDB()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if runtime.GOOS == "windows" {
			var e *os.PathError
			if errors.Is(err, e) {
				return fmt.Errorf("%w: %s", ErrLockedDB, path)
			}
		}
		return fmt.Errorf("%w: %s", err, path)
	}
	return nil
}

// MirrorTmp recursively copies the directory content of src into the hidden tmp mock directory.
func MirrorTmp(src string) error {
	const dirAllAccess fs.FileMode = 0o777
	from, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	tmpDir, err := TempDir()
	if err != nil {
		return err
	}
	return filepath.WalkDir(from, func(path string, d fs.DirEntry, err error) error {
		if path == from {
			return nil
		}
		dest := filepath.Join(tmpDir, strings.Replace(path, from, "", 1))
		if d.IsDir() {
			if errM := os.MkdirAll(dest, dirAllAccess); errM != nil {
				log.Println(errM)
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if _, errC := database.CopyFile(path, dest); errC != nil {
			log.Println(errC)
		}
		return nil
	})
}

// RemoveTmp deletes the hidden tmp mock directory and returns the number of files deleted.
func RemoveTmp() (int, error) {
	tmpDir, err := TempDir()
	if err != nil {
		return 0, err
	}
	count := 0
	err = filepath.WalkDir(tmpDir, func(path string, d fs.DirEntry, err error) error {
		if path == tmpDir {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		count++
		return nil
	})
	if err != nil {
		return 0, err
	}
	return count, os.RemoveAll(tmpDir)
}

// SensenTmp generates 25 subdirectories within the hidden tmp mock directory,
// and copies a mock Windows/DOS .exe program file into one.
// The returned int is the number of bytes copied.
func SensenTmp() (int64, error) {
	tmpDir, err := TempDir()
	if err != nil {
		return 0, err
	}
	n := 0
	dest := ""
	for n < 25 {
		n++
		name := filepath.Join(tmpDir, fmt.Sprintf("mock-dir-%d", n))
		if err = os.MkdirAll(name, PrivateDir); err != nil {
			return 0, err
		}
		if n == 16 {
			dest = name
		}
	}
	item, err := Item(1)
	if err != nil {
		return 0, err
	}
	return database.CopyFile(item, filepath.Join(dest, "some-pretend-windows-app.exe"))
}

// Sum compares b against the expected SHA-256 binary checksum of the test source file item.
func Sum(item int, b [32]byte) (bool, error) {
	if item >= len(checksums) || item < 0 {
		return false, ErrItem
	}
	if checksums[item] == hex.EncodeToString(b[:]) {
		return true, nil
	}
	return false, nil
}

// ItemSum returns the SHA-256 binary checksum of the test source file item.
func ItemSum(item int) (string, error) {
	if item >= len(checksums) || item < 0 {
		return "", ErrItem
	}
	return checksums[item], nil
}
