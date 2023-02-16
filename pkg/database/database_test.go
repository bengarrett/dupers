// © Ben Garrett https://github.com/bengarrett/dupers
package database_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/bengarrett/dupers/internal/mock"
	"github.com/bengarrett/dupers/pkg/database"
	"github.com/stretchr/testify/assert"
)

func TestAll(t *testing.T) {
	results, err := database.All(nil)
	assert.NotNil(t, err)
	assert.Nil(t, results)

	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()

	results, err = database.All(db)
	assert.Nil(t, err)
	const expected = 2
	assert.Equal(t, expected, len(results))
}

func TestClean(t *testing.T) {
	err := database.Clean(nil, true, false)
	assert.NotNil(t, err)

	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()

	err = database.Clean(db, true, false)
	assert.Nil(t, err)
}

func TestCompact(t *testing.T) {
	err := database.Compact(nil, false)
	assert.NotNil(t, err)

	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()

	err = database.Compact(db, false)
	assert.Nil(t, err)
}

func TestCompare(t *testing.T) {
	results, err := database.Compare(nil, "", "")
	assert.NotNil(t, err)
	assert.Nil(t, results)

	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()
	results, err = database.Compare(db, "", "")
	assert.NotNil(t, err)
	assert.Nil(t, results)

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	results, err = database.Compare(db, "", bucket1)
	assert.NotNil(t, err)
	assert.Nil(t, results)

	results, err = database.Compare(db, mock.NoSuchFile, bucket1)
	assert.Nil(t, err)
	assert.NotNil(t, results)
	if results != nil {
		assert.Equal(t, 0, len(*results))
	}

	item, err := mock.Item(1)
	assert.Nil(t, err)
	results, err = database.Compare(db, item, bucket1)
	assert.Nil(t, err)
	assert.NotNil(t, results)
	const expected = 1
	if results != nil {
		assert.Equal(t, expected, len(*results))
	}

	results, err = database.Compare(db, strings.ToUpper(item), bucket1)
	assert.Nil(t, err)
	assert.NotNil(t, results)
	if results != nil {
		assert.Equal(t, 0, len(*results))
	}

	results, err = database.CompareNoCase(db, strings.ToUpper(item), bucket1)
	assert.Nil(t, err)
	assert.NotNil(t, results)
	if results != nil {
		assert.Equal(t, expected, len(*results))
	}
}

func TestCompareBases(t *testing.T) {
	results, err := database.CompareBase(nil, "", "")
	assert.NotNil(t, err)
	assert.Nil(t, results)

	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()
	results, err = database.CompareBase(db, "", "")
	assert.NotNil(t, err)
	assert.Nil(t, results)

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	results, err = database.CompareBase(db, "", bucket1)
	assert.NotNil(t, err)
	assert.Nil(t, results)

	item1, err := mock.Item(1)
	assert.Nil(t, err)
	base := filepath.Base(item1)

	results, err = database.CompareBase(db, base, bucket1)
	assert.Nil(t, err)
	assert.NotNil(t, results)
	const expected = 1
	if results != nil {
		assert.Equal(t, expected, len(*results))
	}

	results, err = database.CompareBase(db, strings.ToUpper(base), bucket1)
	assert.Nil(t, err)
	assert.NotNil(t, results)
	if results != nil {
		assert.Equal(t, 0, len(*results))
	}

	results, err = database.CompareBaseNoCase(db, strings.ToUpper(base), bucket1)
	assert.Nil(t, err)
	assert.NotNil(t, results)
	if results != nil {
		assert.Equal(t, expected, len(*results))
	}
}

func TestExist(t *testing.T) {
	err := database.Exist(nil, "")
	assert.NotNil(t, err)

	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()

	err = database.Exist(db, "")
	assert.NotNil(t, err)

	err = database.Exist(db, mock.NoSuchFile)
	assert.NotNil(t, err)

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	err = database.Exist(db, bucket1)
	assert.Nil(t, err)

	bucket2, err := mock.Bucket(2)
	assert.Nil(t, err)
	err = database.Exist(db, bucket2)
	assert.Nil(t, err)
}

func TestIsEmpty(t *testing.T) {
	err := database.IsEmpty(nil)
	assert.NotNil(t, err)

	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()

	err = database.IsEmpty(db)
	assert.Nil(t, err)

	// delete the buckets from the mock database
	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	bucket2, err := mock.Bucket(2)
	assert.Nil(t, err)
	err = database.RM(db, bucket1)
	assert.Nil(t, err)
	err = database.RM(db, bucket2)
	assert.Nil(t, err)

	err = database.IsEmpty(db)
	assert.NotNil(t, err)
}

func TestList(t *testing.T) {
	results, err := database.List(nil, "")
	assert.NotNil(t, err)
	assert.Equal(t, 0, len(results))

	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()

	_, err = database.List(db, "")
	assert.NotNil(t, err)

	results, err = database.List(db, mock.NoSuchFile)
	assert.NotNil(t, err)
	assert.Equal(t, 0, len(results))

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	results, err = database.List(db, bucket1)
	assert.Nil(t, err)
	const expected = 3
	assert.Equal(t, expected, len(results))
}

func TestInfo(t *testing.T) {
	info, err := database.Info(nil)
	assert.NotNil(t, err)
	assert.Equal(t, "", info)

	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()

	info, err = database.Info(db)
	assert.Nil(t, err)
	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	assert.Contains(t, info, bucket1)
}

func TestRename(t *testing.T) {
	err := database.Rename(nil, "", "")
	assert.NotNil(t, err)

	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()
	err = database.Rename(db, "", "")
	assert.NotNil(t, err)

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	err = database.Rename(db, bucket1, "")
	assert.NotNil(t, err)

	bucket2, err := mock.Bucket(2)
	assert.Nil(t, err)
	err = database.Rename(db, bucket1, bucket2)
	assert.NotNil(t, err, "bucket1 should not be able to be renamed to the existing bucket2")

	err = database.Rename(db, mock.NoSuchFile, bucket2)
	assert.NotNil(t, err)

	err = database.Rename(db, bucket2, mock.NoSuchFile)
	assert.Nil(t, err, "it should be possible to rename bucket to the non-existent bucket name")

	err = database.Rename(db, bucket1, mock.NoSuchFile)
	assert.NotNil(t, err, "the nosuchfile name is already used as a bucket name")
}

func TestCreate(t *testing.T) {
	err := database.Create("")
	assert.NotNil(t, err)

	tmpDir, err := mock.TempDir()
	assert.Nil(t, err)
	path := filepath.Join(tmpDir, "test-create-bbolt-database.db")
	err = database.Create(path)
	assert.Nil(t, err)
	// creating the database again simply reopens and then closes it
	err = database.Create(path)
	assert.Nil(t, err)
}

func TestCheck_DB(t *testing.T) {
	path, err := database.DB()
	assert.Nil(t, err)
	assert.NotEqual(t, "", path)

	i, err := database.Check()
	assert.Nil(t, err)
	assert.Greater(t, i, int64(0))
}
