// © Ben Garrett https://github.com/bengarrett/dupers

// Mock is a set of simulated database and bucket functions for unit testing.
package mock_test

import (
	"os"
	"strings"
	"testing"

	"github.com/bengarrett/dupers/internal/mock"
	"github.com/nalgeon/be"
)

func TestRootDir(t *testing.T) {
	dir, err := os.Stat(mock.RootDir(t))
	be.Equal(t, err, nil)
	be.True(t, dir != nil)
}

func TestBuckets(t *testing.T) {
	bucket1, err := mock.Bucket(t, 1)
	be.Equal(t, err, nil)
	bucket2, err := mock.Bucket(t, 2)
	be.Equal(t, err, nil)
	dir, err := os.Stat(bucket1)
	be.Equal(t, err, nil)
	be.True(t, dir != nil)
	dir, err = os.Stat(bucket2)
	be.Equal(t, err, nil)
	be.True(t, dir != nil)
}

func TestExport(t *testing.T) {
	export1 := mock.Export(t, 1)
	dir, err := os.Stat(export1)
	be.Equal(t, err, nil)
	be.True(t, dir != nil)
	export2 := mock.Export(t, 2)
	dir, err = os.Stat(export2)
	be.Err(t, err)
	be.True(t, dir == nil)
}

func TestNamedDB(t *testing.T) {
	// delete mock db if it exists
	file := mock.NamedDB(t)
	be.True(t, file != "")
	stat, err := os.Stat(file)
	be.Equal(t, err, nil)
	be.True(t, stat != nil)
	// create an empty db for more tests
	path := mock.Create(t)
	be.True(t, path != "")
	stat, err = os.Stat(file)
	be.Equal(t, err, nil)
	be.True(t, stat != nil)
}

func TestDatabase(t *testing.T) {
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)
	p, s := db.Path(), db.String()
	be.True(t, p != "")
	be.True(t, s != "")
}

func TestOpen(t *testing.T) {
	path := mock.Create(t)
	defer os.Remove(path)
	db, _ := mock.Open(t, path)
	defer db.Close()
	p, s := db.Path(), db.String()
	be.True(t, p != "")
	be.True(t, s != "")
}

func TestExtension(t *testing.T) {
	ext := mock.Extension(t, "")
	be.Equal(t, ext, "")
	ext = mock.Extension(t, "arc")
	be.Equal(t, ext, "")
	ext = mock.Extension(t, "7z")
	be.True(t, strings.Contains(ext, "randomfiles.7z"))
}
