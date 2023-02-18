// © Ben Garrett https://github.com/bengarrett/dupers
package bucket_test

import (
	"path/filepath"
	"testing"

	"github.com/bengarrett/dupers/internal/mock"
	"github.com/bengarrett/dupers/pkg/database/internal/bucket"
	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	p := bucket.Parser{}
	items, errs, name, debug := p.Parse(nil)
	assert.Equal(t, 0, items)
	assert.Equal(t, 0, errs)
	assert.Equal(t, "", name)
	assert.Equal(t, false, debug)

	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()

	items, errs, name, debug = p.Parse(db)
	assert.Equal(t, 0, items)
	assert.Equal(t, 0, errs)
	assert.Equal(t, "", name)
	assert.Equal(t, true, debug)

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	p = bucket.Parser{
		Name: bucket1,
	}
	items, errs, name, debug = p.Parse(db)
	assert.Equal(t, 0, items)
	assert.Equal(t, 0, errs)
	assert.Contains(t, name, filepath.Base(bucket1))
	assert.Equal(t, false, debug)

	p = bucket.Parser{
		Name: mock.NoSuchFile,
	}
	items, errs, name, debug = p.Parse(db)
	assert.Equal(t, 0, items)
	assert.Equal(t, 1, errs)
	assert.Equal(t, "", name)
	assert.Equal(t, true, debug)

	item1, err := mock.Item(1)
	assert.Nil(t, err)
	p = bucket.Parser{
		Name: item1,
	}
	items, errs, name, debug = p.Parse(db)
	assert.Equal(t, 0, items)
	assert.Equal(t, 1, errs)
	assert.Equal(t, "", name)
	assert.Equal(t, true, debug)
}

func TestCleaner_Clean(t *testing.T) {
	c := bucket.Cleaner{}
	items, finds, errs, err := c.Clean(nil)
	assert.NotNil(t, err)
	assert.Equal(t, 0, items)
	assert.Equal(t, 0, finds)
	assert.Equal(t, 0, errs)

	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()

	items, finds, errs, err = c.Clean(db)
	assert.Nil(t, err)
	assert.Equal(t, 0, items)
	assert.Equal(t, 0, finds)
	assert.Equal(t, 1, errs)

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	c = bucket.Cleaner{
		Name: bucket1,
	}
	items, finds, errs, err = c.Clean(db)
	assert.Nil(t, err)
	assert.Equal(t, 3, items)
	assert.Equal(t, 0, finds)
	assert.Equal(t, 0, errs)

}

func TestAbs(t *testing.T) {
	s, err := bucket.Abs("")
	assert.NotNil(t, err)
	assert.Equal(t, "", s)

	s, err = bucket.Abs(mock.NoSuchFile)
	assert.Nil(t, err)
	assert.Contains(t, s, mock.NoSuchFile)
}

func TestCount(t *testing.T) {
	val, err := bucket.Count(nil, "")
	assert.NotNil(t, err)
	assert.Equal(t, 0, val)

	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()

	_, err = bucket.Count(db, "")
	assert.NotNil(t, err)

	_, err = bucket.Count(db, mock.NoSuchFile)
	assert.NotNil(t, err)

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	val, err = bucket.Count(db, bucket1)
	assert.Nil(t, err)
	assert.Equal(t, 3, val)

	bucket2, err := mock.Bucket(2)
	assert.Nil(t, err)
	val, err = bucket.Count(db, bucket2)
	assert.Nil(t, err)
	assert.Equal(t, 0, val)
}

func TestStat(t *testing.T) {
	s := bucket.Stat("", false, true)
	assert.Equal(t, "", s)

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	s = bucket.Stat(bucket1, true, true)
	assert.Equal(t, bucket1, s)
}

func TestTotal(t *testing.T) {
	i, err := bucket.Total(nil, nil)
	assert.NotNil(t, err)
	assert.Equal(t, i, 0)

	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()

	i, err = bucket.Total(db, nil)
	assert.NotNil(t, err)
	assert.Equal(t, i, 0)

	i, err = bucket.Total(db, []string{mock.NoSuchFile})
	assert.NotNil(t, err)
	assert.Equal(t, i, 0)

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	bucket2, err := mock.Bucket(2)
	assert.Nil(t, err)

	i, err = bucket.Total(db, []string{bucket1, bucket2})
	assert.Nil(t, err)
	assert.Equal(t, 3, i)
}
