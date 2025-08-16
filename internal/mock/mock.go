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
	"testing"

	"github.com/bengarrett/dupers/pkg/database"
	bolt "go.etcd.io/bbolt"
)

const (
	PrivateFile fs.FileMode = 0o600                 // PrivateFile mode means only the owner has read/write access.
	PrivateDir  fs.FileMode = 0o700                 // PrivateDir mode means only the owner has read/write/dir access.
	SevenZip                = "test/randomfiles.7z" //
	NoSuchFile              = "qwertryuiop"         // NoSuchFile is a non-existent filename.
	filename                = "dupers-*.db"         // filename of the mock database.
	subdir                  = "dupers-mock"         // the sub-directory within config that houses the mock database.
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

func sources() map[int]string {
	return map[int]string{
		0: "/0vlLaUEvzAWP",
		1: "/3a9dnxgSVEnJ",
		2: "/12wZkDDR9CQ0",
	}
}

func checksums() map[int]string {
	return map[int]string{
		0: "1a1d76a3187ccee147e6c807277273afbad5d2680f5eadf1012310743e148f22",
		1: "1bdd103eace1a58d2429d447ac551030a9da424056d2d89a77b1366a04f1f1cc",
		2: "c5f338d4057fb107793032de91b264707c3c27bf9970687a78a080a4bf095c26",
	}
}

func extensions() map[string]string {
	return map[string]string{
		"7z":  "/randomfiles.7z",
		"xz":  "/randomfiles.tar.xz",
		"txt": "/randomfiles.txt",
		"zip": "/randomfiles.zip",
	}
}

const test = "testdata"

var ErrRuntime = errors.New("runtime caller failed")

// Database creates, opens and returns the mock database.
func Database(t *testing.T) (*bolt.DB, string) {
	t.Helper()
	path := Create(t)
	return Open(t, path)
}

// CSV returns the path to a mock exported comma-separated values file.
func CSV(t *testing.T) string {
	t.Helper()
	root := RootDir(t)
	const csv = "export-bucket1.csv"
	return filepath.Join(root, test, csv)
}

// RootDir returns the root directory of the program's source code.
func RootDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Error(ErrRuntime)
		return ""
	}
	return filepath.Join(filepath.Dir(file), "..", "..")
}

// TempDir returns a hidden tmp mock directory path within the
// root directory of the program's source code.
// If the directory doesn't exist, it is created.
func TempDir(t *testing.T) string {
	t.Helper()
	const msg = "mock temporary directory"
	root := RootDir(t)
	tmp, err := os.MkdirTemp(root, ".mock-*") //nolint:usetesting
	if err != nil {
		t.Errorf("%s: %s", msg, err)
	}
	return tmp
}

// Bucket returns the absolute path of test bucket.
func Bucket(t *testing.T, i int) (string, error) {
	t.Helper()
	const msg = "mock bucket path"
	var name string
	const b1, b2, b3 = 1, 2, 3
	switch i {
	case b1:
		name = "bucket1"
	case b2:
		name = "bucket2"
	case b3:
		name = "bucket3"
	default:
		return "", ErrBucket
	}
	path := filepath.Join(RootDir(t), test, name)
	f, err := filepath.Abs(path)
	if err != nil {
		t.Errorf("%s: %e", msg, err)
	}
	if runtime.GOOS == win {
		f = strings.ToLower(f)
	}
	return f, nil
}

// Export returns the absolute path of export csv file for a bucket.
func Export(t *testing.T, i int) string {
	t.Helper()
	const msg = "mock export csv"
	if i >= len(sources()) || i < 0 {
		t.Errorf("%s: %s", msg, ErrItem)
	}
	name := fmt.Sprintf("export-bucket%d.csv", i)
	path := filepath.Join(RootDir(t), test, name)
	f, err := filepath.Abs(path)
	if err != nil {
		t.Errorf("%s: %s", msg, ErrItem)
	}
	return f
}

// Item returns the absolute path of test source file item.
func Item(t *testing.T, i int) string {
	t.Helper()
	const msg = "mock item absolute path"
	if i >= len(sources()) || i < 0 {
		t.Errorf("%s: %s", msg, ErrItem)
	}
	elem := sources()[i]
	bucket1, err := Bucket(t, 1)
	if err != nil {
		t.Errorf("%s: %s", msg, err)
	}
	path := filepath.Join(bucket1, elem)
	f, err := filepath.Abs(path)
	if err != nil {
		t.Errorf("%s: %s", msg, err)
	}

	return f
}

// Extension returns the absolute path of a test file based on an extension.
func Extension(t *testing.T, ext string) string {
	t.Helper()
	const msg = "mock extension"
	elem, ok := extensions()[ext]
	if !ok || elem == "" {
		return ""
	}
	path := filepath.Join(RootDir(t), test, elem)
	f, err := filepath.Abs(path)
	if err != nil {
		t.Errorf("%s: %s", msg, err)
	}
	return f
}

// NamedDB returns the absolute path of a mock Bolt database with a randomly generated filename.
func NamedDB(t *testing.T) string {
	t.Helper()
	const msg = "mock named database"
	dir := t.TempDir()
	path := filepath.Join(dir, subdir)
	if err := os.MkdirAll(path, PrivateDir); err != nil {
		t.Errorf("%s: %s", msg, err)
	}
	f, err := os.CreateTemp(path, filename)
	if err != nil {
		t.Errorf("%s create temp using %q: %s", msg, filename, err)
	}
	defer func() {
		_ = f.Close()
	}()
	return f.Name()
}

// Create the mock database and return its location.
// Note: If this test fails under Windows, try running `go test ./...` after closing VS Code.
// https://github.com/electron-userland/electron-builder/issues/3666
func Create(t *testing.T) string {
	t.Helper()
	const msg = "mock database creation"
	path := NamedDB(t)
	db, err := bolt.Open(path, PrivateFile, nil)
	if err != nil {
		t.Errorf("%s: %s", err, msg)
	}
	defer func() { _ = db.Close() }()
	err = db.Update(func(tx *bolt.Tx) error {
		// delete any existing buckets from the mock database
		if err := tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			return tx.DeleteBucket(name)
		}); err != nil {
			return err
		}
		// create the new mock bucket #1
		bucket1, err := Bucket(t, 1)
		if err != nil {
			return err
		}
		b, err := tx.CreateBucket([]byte(bucket1))
		if err != nil {
			return fmt.Errorf("%w: create bucket: %s", err, bucket1)
		}
		// create the new, but empty mock bucket #2
		const item = 2
		bucket2, err := Bucket(t, item)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucket([]byte(bucket2))
		if err != nil {
			return fmt.Errorf("%w: create bucket: %s", err, bucket1)
		}
		for i := range sources() {
			item := Item(t, i)
			sum256, err := read(item)
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
		t.Errorf("%s: %s", msg, err)
	}
	return path
}

// Read the named file and return its SHA256 checksum.
func read(name string) ([32]byte, error) {
	name = filepath.Clean(name)
	f, err := os.Open(name)
	if err != nil {
		return [32]byte{}, err
	}
	defer func() {
		_ = f.Close()
	}()
	buf := make([]byte, oneMb)
	h := sha256.New()
	if _, err := io.CopyBuffer(h, f, buf); err != nil {
		return [32]byte{}, err
	}
	return [32]byte(h.Sum(nil)), nil
}

// Open the mock database.
// This will need to be closed after use.
func Open(t *testing.T, path string) (*bolt.DB, string) {
	t.Helper()
	const msg = "mock open database"
	db, err := bolt.Open(path, PrivateFile, nil)
	if err != nil {
		t.Errorf("%s: %s", msg, err)
	}
	return db, path
}

// MirrorData recursively copies the directory content of src into the hidden tmp mock directory.
func MirrorData(t *testing.T) string {
	t.Helper()
	const msg = "mock mirror temporary testdata"
	const dirAllAccess fs.FileMode = 0o777
	src := filepath.Join(RootDir(t), test)
	from, err := filepath.Abs(src)
	if err != nil {
		t.Errorf("%s: %s", msg, err)
	}
	tmpDir := t.TempDir()
	err = filepath.WalkDir(from, func(path string, d fs.DirEntry, err error) error {
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
	if err != nil {
		t.Errorf("%s: %s", msg, err)
	}
	return tmpDir
}

// RemoveTmp deletes the hidden tmp mock directory and returns the number of files deleted.
func RemoveTmp(t *testing.T, path string) int {
	t.Helper()
	const msg = "mock remove temporary path"
	count := 0
	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		count++
		return nil
	})
	if err != nil {
		t.Errorf("%s %s", msg, err)
	}
	err = os.RemoveAll(path)
	if err != nil {
		t.Errorf("%s %s", msg, err)
	}
	return count
}

// SensenTmp generates 25 subdirectories within a hidden tmp mock directory,
// and copies a mock Windows/DOS .exe program file into one.
// The returned int is the number of bytes copied.
func SensenTmp(t *testing.T, path string) int64 {
	t.Helper()
	const msg = "mock sensen temporary"
	const expected = 16
	n := 0
	dest := ""
	for n < 25 {
		n++
		name := filepath.Join(path, fmt.Sprintf("mock-dir-%d", n))
		if err := os.MkdirAll(name, PrivateDir); err != nil {
			t.Errorf("%s: %s", msg, err)
		}
		if n == expected {
			dest = name
		}
	}
	item := Item(t, 1)
	i, err := database.CopyFile(item, filepath.Join(dest, "some-pretend-windows-app.exe"))
	if err != nil {
		t.Errorf("%s: %s", msg, err)
	}
	return i
}

// Sum compares b against the expected SHA-256 binary checksum of the test source file item.
func Sum(t *testing.T, item int, b [32]byte) bool {
	t.Helper()
	const msg = "mock sum sha-256 checksum"
	if item >= len(checksums()) || item < 0 {
		t.Errorf("%s %d: %s", msg, item, ErrItem)
	}
	if checksums()[item] == hex.EncodeToString(b[:]) {
		return true
	}
	return false
}

// ItemSum returns the SHA-256 binary checksum of the test source file item.
func ItemSum(t *testing.T, item int) string {
	t.Helper()
	const msg = "mock item sha-256 checksum"
	if item >= len(checksums()) || item < 0 {
		t.Errorf("%s %d: %s", msg, item, ErrItem)
	}
	return checksums()[item]
}
