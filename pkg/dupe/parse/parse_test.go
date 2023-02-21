// © Ben Garrett https://github.com/bengarrett/dupers
package parse_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bengarrett/dupers/internal/mock"
	"github.com/bengarrett/dupers/pkg/database"
	"github.com/bengarrett/dupers/pkg/dupe/parse"
	"github.com/gookit/color"
	"github.com/stretchr/testify/assert"
)

func TestSetBuckets(t *testing.T) {
	db, path, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()
	defer os.Remove(path)

	var s parse.Scanner
	err = s.SetAllBuckets(nil)
	assert.NotNil(t, err)

	err = s.SetAllBuckets(db)
	assert.Nil(t, err)

	const expected = 2
	actual := len(s.Buckets)
	assert.Equal(t, expected, actual)
}

func TestTimer(t *testing.T) {
	p := parse.Scanner{}
	p.SetTimer()
	time.Sleep(100 * time.Millisecond)
	const zero time.Duration = 0
	assert.Greater(t, p.Timer(), zero, "timer should not be 0")
}

func TestParser_SetCompares(t *testing.T) {
	s := parse.Scanner{}
	_, err := s.SetCompares(nil, "")
	assert.NotNil(t, err)

	db, path, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()
	defer os.Remove(path)
	_, err = s.SetCompares(db, "")
	assert.NotNil(t, err)

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	i, err := s.SetCompares(db, parse.Bucket(bucket1))
	assert.Nil(t, err)
	const bucket1Items = 3
	assert.Equal(t, bucket1Items, i)

	bucket2, err := mock.Bucket(2)
	assert.Nil(t, err)
	i, err = s.SetCompares(db, parse.Bucket(bucket2))
	assert.Nil(t, err)
	const bucket2Items = 0 // there's no items in the bucket
	assert.Equal(t, bucket2Items, i)
}

func TestContains(t *testing.T) {
	randm := []string{"weight", "teacher", "budge", "enthusiasm", "familiar"}

	b := parse.Contains("", "")
	assert.Equal(t, true, b)

	b = parse.Contains("", randm...)
	assert.Equal(t, false, b)

	b = parse.Contains("budge", randm...)
	assert.Equal(t, true, b)

	b = parse.Contains("BuDgE", randm...)
	assert.Equal(t, false, b)

	b = parse.Contains("budge.", randm...)
	assert.Equal(t, false, b)

	b = parse.Contains("bud", randm...)
	assert.Equal(t, false, b)
}

func TestExecutable(t *testing.T) {
	b, err := parse.Executable("")
	assert.NotNil(t, err)
	assert.Equal(t, false, b)

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	b, err = parse.Executable(bucket1)
	assert.Nil(t, err)
	assert.Equal(t, false, b)

	item1, err := mock.Item(1)
	assert.Nil(t, err)
	b, err = parse.Executable(item1)
	assert.Nil(t, err)
	assert.Equal(t, false, b)

	tmpDir, err := mock.TempDir()
	assert.Nil(t, err)
	i, err := mock.SensenTmp(tmpDir)
	assert.Nil(t, err)
	assert.Equal(t, int64(20), i)

	b, err = parse.Executable(tmpDir)
	assert.Nil(t, err)
	assert.Equal(t, true, b)

	_, err = mock.RemoveTmp(tmpDir)
	assert.Nil(t, err)
}

func TestRead(t *testing.T) {
	sum, err := parse.Read("")
	assert.NotNil(t, err)
	var empty [32]byte
	assert.Equal(t, empty[:], sum[:])

	item1, err := mock.Item(1)
	assert.Nil(t, err)
	assert.NotEqual(t, "", item1)

	sum, err = parse.Read(item1)
	assert.Nil(t, err)
	ok, err := mock.Sum(1, sum)
	assert.Nil(t, err)
	assert.Equal(t, true, ok)

	item2, err := mock.Item(2)
	assert.Nil(t, err)
	assert.NotEqual(t, "", item1)
	sum, err = parse.Read(item2)
	assert.Nil(t, err)
	ok, err = mock.Sum(2, sum)
	assert.Nil(t, err)
	assert.Equal(t, true, ok)
}

func Test_SetBucket(t *testing.T) {
	s := parse.Scanner{}

	err := s.SetBuckets("")
	assert.Nil(t, err)

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	err = s.SetBuckets(bucket1)
	assert.Nil(t, err)

	count := len(s.Buckets)
	expected := 1
	assert.Equal(t, expected, count)

	bucket2, err := mock.Bucket(2)
	assert.Nil(t, err)
	err = s.SetBuckets(bucket1, bucket2)
	assert.Nil(t, err)

	count = len(s.Buckets)
	expected = 2
	assert.Equal(t, expected, count)

	b := s.BucketS()
	assert.Contains(t, b, "bucket1")
	assert.Contains(t, b, "bucket2")
}

func Test_SetSource(t *testing.T) {
	s := parse.Scanner{}
	err := s.SetSource("")
	assert.NotNil(t, err)

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)

	err = s.SetSource(bucket1)
	assert.Nil(t, err)

	got := s.GetSource()
	assert.Equal(t, bucket1, got)
}

func TestMarker(t *testing.T) {
	color.Enable = false
	s := parse.Marker("", "", false)
	assert.Equal(t, "", s)

	item1, err := mock.Item(1)
	assert.Nil(t, err)
	assert.NotEqual(t, "", item1)

	file := database.Filepath(item1)
	s = parse.Marker(file, "", false)
	assert.Equal(t, item1, s)

	term := filepath.Base(item1)
	s = parse.Marker(file, term, false)
	assert.Equal(t, item1, s)

	s = parse.Marker(file, term, true)
	assert.Equal(t, item1, s)
}

func TestPrint(t *testing.T) {
	m := make(database.Matches)
	s := parse.Print(false, false, "", &m)
	assert.Equal(t, "", s)

	item1, err := mock.Item(1)
	assert.Nil(t, err)
	sum1, err := mock.ItemSum(1)
	assert.Nil(t, err)
	m[database.Filepath(item1)] = database.Bucket(sum1) // 1 match
	s = parse.Print(false, false, "", &m)
	assert.Contains(t, s, item1)
	assert.Contains(t, s, sum1)

	s = parse.Print(true, false, "", &m)
	assert.Contains(t, s, item1)
	assert.NotContains(t, s, sum1)

	// exact and term are untested as they only effect ANSI color output.
}
