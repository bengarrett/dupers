// © Ben Garrett https://github.com/bengarrett/dupers
package database_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bengarrett/dupers/internal/mock"
	"github.com/bengarrett/dupers/pkg/database"
	"github.com/stretchr/testify/assert"
)

func TestBackup(t *testing.T) {
	name, written, err := database.Backup()
	assert.Nil(t, err)
	assert.NotEqual(t, "", name)
	defer os.Remove(name)
	assert.Greater(t, written, int64(0))
}

func TestCopyFile(t *testing.T) {
	written, err := database.CopyFile("", "")
	assert.NotNil(t, err)
	assert.Equal(t, int64(0), written)

	item1, err := mock.Item(1)
	assert.Nil(t, err)

	written, err = database.CopyFile(item1, "")
	assert.NotNil(t, err)
	assert.Equal(t, int64(0), written)

	tmpDir, err := mock.TempDir()
	assert.Nil(t, err)
	defer mock.RemoveTmp()

	const expected = int64(20)
	dest := filepath.Join(tmpDir, "some-random-file.stuff")
	written, err = database.CopyFile(item1, dest)
	assert.Nil(t, err)
	assert.Equal(t, expected, written)
}

func TestCSVExport(t *testing.T) {
	csv, err := database.CSVExport(nil, "")
	assert.NotNil(t, err)
	assert.Equal(t, "", csv)

	db, path, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()
	defer os.Remove(path)

	csv, err = database.CSVExport(db, "")
	assert.NotNil(t, err)
	assert.Equal(t, "", csv)

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	assert.NotEqual(t, "", bucket1)

	csv, err = database.CSVExport(db, bucket1)
	assert.Nil(t, err)
	assert.NotEqual(t, "", bucket1)

	info, err := os.Stat(csv)
	assert.Nil(t, err)
	defer os.Remove(csv)
	assert.Greater(t, info.Size(), int64(0))
}

func TestCSVImport(t *testing.T) {
	imported, err := database.CSVImport(nil, "", false)
	assert.NotNil(t, err)
	assert.Equal(t, 0, imported)

	db, path, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()
	defer os.Remove(path)

	imported, err = database.CSVImport(db, "", false)
	assert.NotNil(t, err)
	assert.Equal(t, 0, imported)

	imported, err = database.CSVImport(db, mock.NoSuchFile, false)
	assert.NotNil(t, err)
	assert.Equal(t, 0, imported)

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	assert.NotEqual(t, "", bucket1)

	csv, err := database.CSVExport(db, bucket1)
	assert.Nil(t, err)
	assert.NotEqual(t, "", bucket1)
	defer os.Remove(csv)

	imported, err = database.CSVImport(db, csv, false)
	assert.Nil(t, err)
	assert.Equal(t, 3, imported)
}

func TestImport(t *testing.T) {
	export1, err := mock.Export(1)
	assert.Nil(t, err)
	assert.NotEqual(t, "", export1)

	db, path, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()
	defer os.Remove(path)

	imported, err := database.Import(db, "", nil)
	assert.NotNil(t, err)
	assert.Equal(t, 0, imported)

	bucket2, err := mock.Bucket(2)
	assert.Nil(t, err)
	assert.NotEqual(t, "", bucket2)

	imported, err = database.Import(db, database.Bucket(bucket2), nil)
	assert.NotNil(t, err)
	assert.Equal(t, 0, imported)

	csv, err := os.Open(export1)
	assert.Nil(t, err)
	defer csv.Close()

	bucket, ls, err := database.Scanner(csv)
	assert.Nil(t, err)
	assert.NotNil(t, ls)
	assert.NotEqual(t, "", bucket)
	const expected = 26
	if ls != nil {
		assert.Equal(t, len(*ls), expected)
	}

	imported, err = database.Import(db, database.Bucket(bucket2), ls)
	assert.Nil(t, err)
	assert.Equal(t, expected, imported)
}

func TestOpenRead(t *testing.T) {
	db, err := database.OpenRead()
	assert.Nil(t, err)
	defer db.Close()
	assert.NotNil(t, db)
	assert.Equal(t, true, db.IsReadOnly())
}

func TestOpenWrite(t *testing.T) {
	db, err := database.OpenWrite()
	assert.Nil(t, err)
	defer db.Close()
	assert.NotNil(t, db)
	assert.Equal(t, false, db.IsReadOnly())
}

func TestUsage(t *testing.T) {
	s, err := database.Usage(nil, "", true)
	assert.NotNil(t, err)
	assert.Equal(t, "", s)

	db, path, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()
	defer os.Remove(path)

	s, err = database.Usage(db, "", true)
	assert.NotNil(t, err)
	assert.Equal(t, "", s)

	bucket2, err := mock.Bucket(2)
	assert.Nil(t, err)
	assert.NotEqual(t, "", bucket2)
	s, err = database.Usage(db, bucket2, true)
	assert.Nil(t, err)
	assert.Contains(t, s, bucket2)
}
