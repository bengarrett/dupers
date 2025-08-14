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

func TestBackup(t *testing.T) {
	name, written, err := database.Backup()
	be.Err(t, err, nil)
	be.True(t, name != "")
	defer os.Remove(name)
	be.True(t, written > 0)
}

func TestCopyFile(t *testing.T) {
	written, err := database.CopyFile("", "")
	be.Err(t, err)
	be.Equal(t, int64(0), written)
	item1, err := mock.Item(1)
	be.Err(t, err, nil)
	written, err = database.CopyFile(item1, "")
	be.Err(t, err)
	be.Equal(t, int64(0), written)
	tmpDir, err := mock.TempDir()
	be.Err(t, err, nil)
	defer mock.RemoveTmp(tmpDir)
	const expected = int64(20)
	dest := filepath.Join(tmpDir, "some-random-file.stuff")
	written, err = database.CopyFile(item1, dest)
	be.Err(t, err, nil)
	be.Equal(t, expected, written)
}

func TestCSVExport(t *testing.T) {
	csv, err := database.CSVExport(nil, "")
	be.Err(t, err)
	be.Equal(t, "", csv)
	db, path, err := mock.Database()
	be.Err(t, err, nil)
	defer db.Close()
	defer os.Remove(path)
	csv, err = database.CSVExport(db, "")
	be.Err(t, err)
	be.Equal(t, "", csv)
	bucket1, err := mock.Bucket(1)
	be.Err(t, err, nil)
	be.True(t, bucket1 != "")
	csv, err = database.CSVExport(db, bucket1)
	be.Err(t, err, nil)
	be.True(t, bucket1 != "")
	info, err := os.Stat(csv)
	be.Err(t, err, nil)
	defer os.Remove(csv)
	be.True(t, info.Size() > 0)
}

func TestCSVImport(t *testing.T) {
	imported, err := database.CSVImport(nil, "", false)
	be.Err(t, err)
	db, path, err := mock.Database()
	be.Err(t, err, nil)
	defer db.Close()
	defer os.Remove(path)
	imported, err = database.CSVImport(db, "", false)
	be.Err(t, err)
	be.Equal(t, 0, imported)
	imported, err = database.CSVImport(db, mock.NoSuchFile, false)
	be.Err(t, err)
	be.Equal(t, 0, imported)
	bucket1, err := mock.Bucket(1)
	be.Err(t, err, nil)
	be.True(t, bucket1 != "")
	csv, err := database.CSVExport(db, bucket1)
	be.Err(t, err, nil)
	be.True(t, bucket1 != "")
	defer os.Remove(csv)
	imported, err = database.CSVImport(db, csv, false)
	be.Err(t, err, nil)
	be.Equal(t, 3, imported)
}

func TestImport(t *testing.T) {
	export1, err := mock.Export(1)
	be.Err(t, err, nil)
	be.True(t, export1 != "")
	db, path, err := mock.Database()
	be.Err(t, err, nil)
	defer db.Close()
	defer os.Remove(path)
	imported, err := database.Import(db, "", nil)
	be.Err(t, err)
	be.Equal(t, 0, imported)
	bucket2, err := mock.Bucket(2)
	be.Err(t, err, nil)
	be.True(t, bucket2 != "")
	imported, err = database.Import(db, database.Bucket(bucket2), nil)
	be.Err(t, err)
	be.Equal(t, 0, imported)
	csv, err := os.Open(export1)
	be.Err(t, err, nil)
	defer csv.Close()
	bucket, ls, err := database.Scanner(csv)
	be.Err(t, err, nil)
	be.True(t, ls != nil)
	be.True(t, bucket != "")
	const expected = 26
	if ls != nil {
		be.Equal(t, len(*ls), expected)
	}
	imported, err = database.Import(db, database.Bucket(bucket2), ls)
	be.Err(t, err, nil)
	be.Equal(t, expected, imported)
}

func TestOpenRead(t *testing.T) {
	db, err := database.OpenRead()
	be.Err(t, err, nil)
	defer db.Close()
	be.True(t, db != nil)
	be.Equal(t, true, db.IsReadOnly())
}

func TestOpenWrite(t *testing.T) {
	db, err := database.OpenWrite()
	be.Err(t, err, nil)
	defer db.Close()
	be.True(t, db != nil)
	be.Equal(t, false, db.IsReadOnly())
}

func TestUsage(t *testing.T) {
	s, err := database.Usage(nil, "", true)
	be.Err(t, err)
	be.Equal(t, "", s)
	db, path, err := mock.Database()
	be.Err(t, err, nil)
	defer db.Close()
	defer os.Remove(path)
	s, err = database.Usage(db, "", true)
	be.Err(t, err)
	be.Equal(t, "", s)
	bucket2, err := mock.Bucket(2)
	be.Err(t, err, nil)
	be.True(t, bucket2 != "")
	s, err = database.Usage(db, bucket2, true)
	be.Err(t, err, nil)
	ok := strings.Contains(s, bucket2)
	be.True(t, ok)
}
