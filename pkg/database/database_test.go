// © Ben Garrett https://github.com/bengarrett/dupers
package database_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bengarrett/dupers/internal/mock"
	"github.com/bengarrett/dupers/pkg/database"
	"github.com/nalgeon/be"
)

func TestAll(t *testing.T) {
	results, err := database.All(nil)
	be.Err(t, err)
	be.Equal(t, results, nil)
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)
	results, err = database.All(db)
	be.Err(t, err, nil)
	const expected = 2
	be.Equal(t, len(results), expected)
}

func TestClean(t *testing.T) {
	err := database.Clean(nil, true, false)
	be.Err(t, err)
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)
	err = database.Clean(db, true, false)
	be.Err(t, err, nil)
}

func TestCompact(t *testing.T) {
	err := database.Compact(nil, false)
	be.Err(t, err)
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)
	err = database.Compact(db, false)
	be.Err(t, err, nil)
}

func TestCompare(t *testing.T) {
	results, err := database.Compare(nil, "", "")
	be.Err(t, err)
	be.Equal(t, results, nil)
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)
	results, err = database.Compare(db, "", "")
	be.Err(t, err)
	be.Equal(t, results, nil)
	bucket1, err := mock.Bucket(t, 1)
	be.Err(t, err, nil)
	results, err = database.Compare(db, "", bucket1)
	be.Err(t, err)
	be.Equal(t, results, nil)
	results, err = database.Compare(db, mock.NoSuchFile, bucket1)
	be.Err(t, err, nil)
	be.True(t, results != nil)
	if results != nil {
		be.Equal(t, len(*results), 0)
	}
	item, err := mock.Item(t, 1)
	be.Err(t, err, nil)
	results, err = database.Compare(db, item, bucket1)
	be.Err(t, err, nil)
	be.True(t, results != nil)
	const expected = 1
	if results != nil {
		be.Equal(t, len(*results), expected)
	}
	results, err = database.Compare(db, strings.ToUpper(item), bucket1)
	be.Err(t, err, nil)
	be.True(t, results != nil)
	if results != nil {
		be.Equal(t, len(*results), 0)
	}
	results, err = database.CompareNoCase(db, strings.ToUpper(item), bucket1)
	be.Err(t, err, nil)
	be.True(t, results != nil)
	if results != nil {
		be.Equal(t, len(*results), expected)
	}
}

func TestCompareBases(t *testing.T) {
	results, err := database.CompareBase(nil, "", "")
	be.Err(t, err)
	be.Equal(t, results, nil)
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)
	results, err = database.CompareBase(db, "", "")
	be.Err(t, err)
	be.Equal(t, results, nil)
	bucket1, err := mock.Bucket(t, 1)
	be.Err(t, err, nil)
	results, err = database.CompareBase(db, "", bucket1)
	be.Err(t, err)
	be.Equal(t, results, nil)
	item1, err := mock.Item(t, 1)
	be.Err(t, err, nil)
	base := filepath.Base(item1)
	results, err = database.CompareBase(db, base, bucket1)
	be.Err(t, err, nil)
	be.True(t, results != nil)
	const expected = 1
	if results != nil {
		be.Equal(t, expected, len(*results))
	}
	results, err = database.CompareBase(db, strings.ToUpper(base), bucket1)
	be.Err(t, err, nil)
	be.True(t, results != nil)
	if results != nil {
		be.Equal(t, 0, len(*results))
	}
	results, err = database.CompareBaseNoCase(db, strings.ToUpper(base), bucket1)
	be.Err(t, err, nil)
	be.True(t, results != nil)
	if results != nil {
		be.Equal(t, expected, len(*results))
	}
}

func TestExist(t *testing.T) {
	err := database.Exist(nil, "")
	be.Err(t, err)
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)
	err = database.Exist(db, "")
	be.Err(t, err)
	err = database.Exist(db, mock.NoSuchFile)
	be.Err(t, err)
	bucket1, err := mock.Bucket(t, 1)
	be.Err(t, err, nil)
	err = database.Exist(db, bucket1)
	be.Err(t, err, nil)
	bucket2, err := mock.Bucket(t, 2)
	be.Err(t, err, nil)
	err = database.Exist(db, bucket2)
	be.Err(t, err, nil)
}

func TestIsEmpty(t *testing.T) {
	err := database.IsEmpty(nil)
	be.Err(t, err)
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)
	err = database.IsEmpty(db)
	be.Err(t, err, nil)
	// delete the buckets from the mock database
	bucket1, err := mock.Bucket(t, 1)
	be.Err(t, err, nil)
	bucket2, err := mock.Bucket(t, 2)
	be.Err(t, err, nil)
	err = database.Remove(db, bucket1)
	be.Err(t, err, nil)
	err = database.Remove(db, bucket2)
	be.Err(t, err, nil)
	err = database.IsEmpty(db)
	be.Err(t, err)
}

func TestList(t *testing.T) {
	results, err := database.List(nil, "")
	be.Err(t, err)
	be.Equal(t, 0, len(results))
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)
	_, err = database.List(db, "")
	be.Err(t, err)
	results, err = database.List(db, mock.NoSuchFile)
	be.Err(t, err)
	be.Equal(t, 0, len(results))
	bucket1, err := mock.Bucket(t, 1)
	be.Err(t, err, nil)
	results, err = database.List(db, bucket1)
	be.Err(t, err, nil)
	const expected = 3
	be.Equal(t, expected, len(results))
}

func TestInfo(t *testing.T) {
	info, err := database.Info(nil)
	be.Err(t, err)
	be.Equal(t, "", info)
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)
	info, err = database.Info(db)
	be.Err(t, err, nil)
	bucket1, err := mock.Bucket(t, 1)
	be.Err(t, err, nil)
	ok := strings.Contains(info, bucket1)
	be.True(t, ok)
}

func TestRename(t *testing.T) {
	err := database.Rename(nil, "", "")
	be.Err(t, err)
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)
	err = database.Rename(db, "", "")
	be.Err(t, err)
	bucket1, err := mock.Bucket(t, 1)
	be.Err(t, err, nil)
	err = database.Rename(db, bucket1, "")
	be.Err(t, err)
	bucket2, err := mock.Bucket(t, 2)
	be.Err(t, err, nil)
	err = database.Rename(db, bucket1, bucket2)
	be.Err(t, err)
	err = database.Rename(db, mock.NoSuchFile, bucket2)
	be.Err(t, err)
	err = database.Rename(db, bucket2, mock.NoSuchFile)
	be.Err(t, err, nil)
	err = database.Rename(db, bucket1, mock.NoSuchFile)
	be.Err(t, err)
}

func TestCreate(t *testing.T) {
	err := database.Create("")
	be.Err(t, err)
	tmpDir := mock.TempDir(t)
	path := filepath.Join(tmpDir, "test-create-bbolt-database.db")
	err = database.Create(path)
	be.Err(t, err, nil)
	// creating the database again simply reopens and then closes it
	err = database.Create(path)
	be.Err(t, err, nil)
}

func TestCheck_DB(t *testing.T) {
	path, err := database.DB()
	be.Err(t, err, nil)
	be.True(t, path != "")
	i, err := database.Check()
	be.Err(t, err, nil)
	be.True(t, i > 0)
}
