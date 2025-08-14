// © Ben Garrett https://github.com/bengarrett/dupers
package parse_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bengarrett/dupers/internal/mock"
	"github.com/bengarrett/dupers/pkg/database"
	"github.com/bengarrett/dupers/pkg/dupe/parse"
	"github.com/gookit/color"
	"github.com/nalgeon/be"
)

func TestSetBuckets(t *testing.T) {
	db, path, err := mock.Database()
	be.Err(t, err, nil)
	defer db.Close()
	defer os.Remove(path)
	var s parse.Scanner
	err = s.SetAllBuckets(nil)
	be.Err(t, err)
	err = s.SetAllBuckets(db)
	be.Err(t, err, nil)
	const expected = 2
	actual := len(s.Buckets)
	be.Equal(t, expected, actual)
}

func TestTimer(t *testing.T) {
	p := parse.Scanner{}
	p.SetTimer()
	time.Sleep(100 * time.Millisecond)
	const zero time.Duration = 0
	be.True(t, p.Timer() > zero)
}

func TestParser_SetCompares(t *testing.T) {
	s := parse.Scanner{}
	_, err := s.SetCompares(nil, "")
	be.Err(t, err)
	db, path, err := mock.Database()
	be.Err(t, err, nil)
	defer db.Close()
	defer os.Remove(path)
	_, err = s.SetCompares(db, "")
	be.Err(t, err)
	bucket1, err := mock.Bucket(1)
	be.Err(t, err, nil)
	i, err := s.SetCompares(db, parse.Bucket(bucket1))
	be.Err(t, err, nil)
	const bucket1Items = 3
	be.Equal(t, bucket1Items, i)
	bucket2, err := mock.Bucket(2)
	be.Err(t, err, nil)
	i, err = s.SetCompares(db, parse.Bucket(bucket2))
	be.Err(t, err, nil)
	const bucket2Items = 0 // there's no items in the bucket
	be.Equal(t, bucket2Items, i)
}

func TestContains(t *testing.T) {
	randm := []string{"weight", "teacher", "budge", "enthusiasm", "familiar"}
	b := parse.Contains("", "")
	be.Equal(t, true, b)
	b = parse.Contains("", randm...)
	be.Equal(t, false, b)
	b = parse.Contains("budge", randm...)
	be.Equal(t, true, b)
	b = parse.Contains("BuDgE", randm...)
	be.Equal(t, false, b)
	b = parse.Contains("budge.", randm...)
	be.Equal(t, false, b)
	b = parse.Contains("bud", randm...)
	be.Equal(t, false, b)
}

func TestExecutable(t *testing.T) {
	b, err := parse.Executable("")
	be.Err(t, err)
	be.Equal(t, false, b)
	bucket1, err := mock.Bucket(1)
	be.Err(t, err, nil)
	b, err = parse.Executable(bucket1)
	be.Err(t, err, nil)
	be.Equal(t, false, b)
	item1, err := mock.Item(1)
	be.Err(t, err, nil)
	b, err = parse.Executable(item1)
	be.Err(t, err, nil)
	be.Equal(t, false, b)
	tmpDir, err := mock.TempDir()
	be.Err(t, err, nil)
	i, err := mock.SensenTmp(tmpDir)
	be.Err(t, err, nil)
	be.Equal(t, int64(20), i)
	b, err = parse.Executable(tmpDir)
	be.Err(t, err, nil)
	be.Equal(t, true, b)
	_, err = mock.RemoveTmp(tmpDir)
	be.Err(t, err, nil)
}

func TestRead(t *testing.T) {
	sum, err := parse.Read("")
	be.Err(t, err)
	var empty [32]byte
	be.Equal(t, empty[:], sum[:])
	item1, err := mock.Item(1)
	be.Err(t, err, nil)
	be.True(t, item1 != "")
	sum, err = parse.Read(item1)
	be.Err(t, err, nil)
	ok, err := mock.Sum(1, sum)
	be.Err(t, err, nil)
	be.Equal(t, true, ok)
	item2, err := mock.Item(2)
	be.Err(t, err, nil)
	be.True(t, item1 != "")
	sum, err = parse.Read(item2)
	be.Err(t, err, nil)
	ok, err = mock.Sum(2, sum)
	be.Err(t, err, nil)
	be.Equal(t, true, ok)
}

func Test_SetBucket(t *testing.T) {
	s := parse.Scanner{}
	err := s.SetBuckets("")
	be.Err(t, err, nil)
	bucket1, err := mock.Bucket(1)
	be.Err(t, err, nil)
	err = s.SetBuckets(bucket1)
	be.Err(t, err, nil)
	count := len(s.Buckets)
	expected := 1
	be.Equal(t, expected, count)
	bucket2, err := mock.Bucket(2)
	be.Err(t, err, nil)
	err = s.SetBuckets(bucket1, bucket2)
	be.Err(t, err, nil)
	count = len(s.Buckets)
	expected = 2
	be.Equal(t, expected, count)
	b := s.BucketS()
	ok := strings.Contains(b, "bucket1")
	be.True(t, ok)
	ok = strings.Contains(b, "bucket2")
	be.True(t, ok)
}

func Test_SetSource(t *testing.T) {
	s := parse.Scanner{}
	err := s.SetSource("")
	be.Err(t, err)
	bucket1, err := mock.Bucket(1)
	be.Err(t, err, nil)
	err = s.SetSource(bucket1)
	be.Err(t, err, nil)
	got := s.GetSource()
	be.Equal(t, bucket1, got)
}

func TestMarker(t *testing.T) {
	color.Enable = false
	s := parse.Marker("", "", false)
	be.Equal(t, "", s)
	item1, err := mock.Item(1)
	be.Err(t, err, nil)
	be.True(t, item1 != "")
	file := database.Filepath(item1)
	s = parse.Marker(file, "", false)
	be.Equal(t, item1, s)
	term := filepath.Base(item1)
	s = parse.Marker(file, term, false)
	be.Equal(t, item1, s)
	s = parse.Marker(file, term, true)
	be.Equal(t, item1, s)
}

func TestPrint(t *testing.T) {
	m := make(database.Matches)
	s := parse.Print(false, false, "", &m)
	be.Equal(t, "", s)
	item1, err := mock.Item(1)
	be.Err(t, err, nil)
	sum1, err := mock.ItemSum(1)
	be.Err(t, err, nil)
	m[database.Filepath(item1)] = database.Bucket(sum1) // 1 match
	s = parse.Print(false, false, "", &m)
	ok := strings.Contains(s, item1)
	be.True(t, !ok)
	ok = strings.Contains(s, sum1)
	be.True(t, ok)
	s = parse.Print(true, false, "", &m)
	ok = strings.Contains(s, item1)
	be.True(t, ok)
	ok = strings.Contains(s, sum1)
	be.True(t, !ok)
	// exact and term are untested as they only effect ANSI color output.
}
