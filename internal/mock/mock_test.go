// © Ben Garrett https://github.com/bengarrett/dupers

// Mock is a set of simulated database and bucket functions for unit testing.
package mock_test

import (
	"os"
	"testing"

	"github.com/bengarrett/dupers/internal/mock"
	"github.com/stretchr/testify/assert"
)

func TestRootDir(t *testing.T) {
	dir, err := os.Stat(mock.RootDir())
	assert.Nil(t, err)
	assert.NotEqual(t, "", dir)
}

func TestBuckets(t *testing.T) {
	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	bucket2, err := mock.Bucket(2)
	assert.Nil(t, err)
	dir, err := os.Stat(bucket1)
	assert.Nil(t, err)
	assert.NotEqual(t, "", dir)
	dir, err = os.Stat(bucket2)
	assert.Nil(t, err)
	assert.NotEqual(t, "", dir)
}

func TestExport(t *testing.T) {
	export1, err := mock.Export(1)
	assert.Nil(t, err)
	dir, err := os.Stat(export1)
	assert.Nil(t, err)
	assert.NotEqual(t, "", dir)
	export2, err := mock.Export(2)
	assert.Nil(t, err)
	dir, err = os.Stat(export2)
	assert.NotNil(t, err)
	assert.Nil(t, dir)
}

func TestNamedDB(t *testing.T) {
	// delete mock db if it exists
	err := mock.Delete()
	assert.Nil(t, err)
	file, err := mock.NamedDB()
	assert.Nil(t, err)
	assert.NotEqual(t, "", file)
	stat, err := os.Stat(file)
	assert.NotNil(t, err, "expected an error for a non-existent database file")
	assert.Nil(t, stat)
	// create an empty db for more tests
	path := ""
	path, err = mock.Create()
	defer mock.Delete()
	assert.Nil(t, err, "expected a database at: "+file)
	assert.NotEqual(t, "", path)
	stat, err = os.Stat(file)
	assert.Nil(t, err)
	assert.NotNil(t, stat)
}

func TestDatabase(t *testing.T) {
	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()
	assert.NotEqual(t, "", db.Path())
	assert.NotEqual(t, "", db.String())
}

func TestOpen(t *testing.T) {
	db, err := mock.Open()
	assert.Nil(t, err)
	defer db.Close()
	assert.NotEqual(t, "", db.Path())
	assert.NotEqual(t, "", db.String())
}

func TestExtension(t *testing.T) {
	ext, err := mock.Extension("")
	assert.NotNil(t, err)
	assert.Equal(t, "", ext)

	ext, err = mock.Extension("arc")
	assert.NotNil(t, err)
	assert.Equal(t, "", ext)

	ext, err = mock.Extension("7z")
	assert.Nil(t, err)
	assert.Contains(t, ext, "randomfiles.7z")
}
