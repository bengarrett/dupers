// © Ben Garrett https://github.com/bengarrett/dupers
package bucket_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bengarrett/dupers/internal/mock"
	"github.com/bengarrett/dupers/pkg/database/bucket"
	"github.com/nalgeon/be"
)

func TestParse(t *testing.T) {
	p := bucket.Parser{}
	items, errs, name, debug := p.Parse(nil)
	fmt.Fprintln(os.Stderr, items, errs, name, debug)
	be.Equal(t, items, -1)
	be.Equal(t, errs, -1)
	be.Equal(t, name, "")
	be.True(t, debug)

	db, path, err := mock.Database()
	be.Err(t, err, nil)
	defer db.Close()
	defer os.Remove(path)

	items, errs, name, debug = p.Parse(db)
	be.Equal(t, items, 0)
	be.Equal(t, errs, 0)
	be.Equal(t, name, "")
	be.True(t, debug)

	bucket1, err := mock.Bucket(1)
	be.Err(t, err, nil)
	p = bucket.Parser{
		Name: bucket1,
	}
	items, errs, name, debug = p.Parse(db)
	be.Equal(t, items, 0)
	be.Equal(t, errs, 0)
	be.True(t, strings.Contains(name, filepath.Base(bucket1)))
	be.True(t, !debug)

	p = bucket.Parser{
		Name: mock.NoSuchFile,
	}
	items, errs, name, debug = p.Parse(db)
	be.Equal(t, items, 0)
	be.Equal(t, errs, 1)
	be.Equal(t, name, "")
	be.True(t, debug)

	item1, err := mock.Item(1)
	be.Err(t, err, nil)
	p = bucket.Parser{
		Name: item1,
	}
	items, errs, name, debug = p.Parse(db)
	be.Equal(t, items, 0)
	be.Equal(t, errs, 1)
	be.Equal(t, name, "")
	be.True(t, debug)
}

func TestCleaner_Clean(t *testing.T) {
	c := bucket.Cleaner{}
	items, finds, errs, err := c.Clean(nil)
	be.Err(t, err)
	be.Equal(t, items, 0)
	be.Equal(t, finds, 0)
	be.Equal(t, errs, 0)

	db, path, err := mock.Database()
	be.Err(t, err, nil)
	defer db.Close()
	defer os.Remove(path)

	items, finds, errs, err = c.Clean(db)
	be.Err(t, err, nil)
	be.Equal(t, items, 0)
	be.Equal(t, finds, 0)
	be.Equal(t, errs, 1)

	bucket1, err := mock.Bucket(1)
	be.Err(t, err, nil)
	c = bucket.Cleaner{
		Name: bucket1,
	}
	items, finds, errs, err = c.Clean(db)
	be.Err(t, err, nil)
	be.Equal(t, items, 3)
	be.Equal(t, finds, 0)
	be.Equal(t, errs, 0)
}

func TestAbs(t *testing.T) {
	s, err := bucket.Abs("")
	be.Err(t, err)
	be.Equal(t, s, "")

	s, err = bucket.Abs(mock.NoSuchFile)
	be.Err(t, err, nil)
	be.True(t, strings.Contains(s, mock.NoSuchFile))
}

func TestCount(t *testing.T) {
	val, err := bucket.Count(nil, "")
	be.Err(t, err)
	be.Equal(t, val, 0)

	db, path, err := mock.Database()
	be.Err(t, err, nil)
	defer db.Close()
	defer os.Remove(path)

	_, err = bucket.Count(db, "")
	be.Err(t, err)

	_, err = bucket.Count(db, mock.NoSuchFile)
	be.Err(t, err)

	bucket1, err := mock.Bucket(1)
	be.Err(t, err, nil)
	val, err = bucket.Count(db, bucket1)
	be.Err(t, err, nil)
	be.Equal(t, val, 3)

	bucket2, err := mock.Bucket(2)
	be.Err(t, err, nil)
	val, err = bucket.Count(db, bucket2)
	be.Err(t, err, nil)
	be.Equal(t, val, 0)
}

func TestStat(t *testing.T) {
	s := bucket.Stat("", false, true)
	be.Equal(t, s, "")

	bucket1, err := mock.Bucket(1)
	be.Err(t, err, nil)
	s = bucket.Stat(bucket1, true, true)
	be.Equal(t, s, bucket1)
}

func TestTotal(t *testing.T) {
	i, err := bucket.Total(nil, nil)
	be.Err(t, err)
	be.Equal(t, i, 0)

	db, path, err := mock.Database()
	be.Err(t, err, nil)
	defer db.Close()
	defer os.Remove(path)

	i, err = bucket.Total(db, nil)
	be.Err(t, err)
	be.Equal(t, i, 0)

	i, err = bucket.Total(db, []string{mock.NoSuchFile})
	be.Err(t, err)
	be.Equal(t, i, 0)

	bucket1, err := mock.Bucket(1)
	be.Err(t, err, nil)
	bucket2, err := mock.Bucket(2)
	be.Err(t, err, nil)

	i, err = bucket.Total(db, []string{bucket1, bucket2})
	be.Err(t, err, nil)
	be.Equal(t, i, 3)
}
