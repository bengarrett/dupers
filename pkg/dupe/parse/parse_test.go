// © Ben Garrett https://github.com/bengarrett/dupers
package parse_test

import (
	"bytes"
	"fmt"
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
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)
	var s parse.Scanner
	err := s.SetAllBuckets(nil)
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
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)
	_, err = s.SetCompares(db, "")
	be.Err(t, err)
	bucket1, err := mock.Bucket(t, 1)
	be.Err(t, err, nil)
	i, err := s.SetCompares(db, parse.Bucket(bucket1))
	be.Err(t, err, nil)
	const bucket1Items = 3
	be.Equal(t, bucket1Items, i)
	bucket2, err := mock.Bucket(t, 2)
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
	bucket1, err := mock.Bucket(t, 1)
	be.Err(t, err, nil)
	b, err = parse.Executable(bucket1)
	be.Err(t, err, nil)
	be.Equal(t, false, b)
	item1 := mock.Item(t, 1)
	be.Err(t, err, nil)
	b, err = parse.Executable(item1)
	be.Err(t, err, nil)
	be.Equal(t, false, b)
	tmpDir := t.TempDir()
	i := mock.SensenTmp(t, tmpDir)
	be.Equal(t, int64(20), i)
	b, err = parse.Executable(tmpDir)
	be.Err(t, err, nil)
	be.Equal(t, true, b)
}

func TestRead(t *testing.T) {
	sum, err := parse.Read("")
	be.Err(t, err)
	var empty [32]byte
	be.Equal(t, empty[:], sum[:])
	item1 := mock.Item(t, 1)
	be.True(t, item1 != "")
	sum, err = parse.Read(item1)
	be.Err(t, err, nil)
	ok := mock.Sum(t, 1, sum)
	be.Equal(t, true, ok)
	item2 := mock.Item(t, 2)
	be.True(t, item1 != "")
	sum, err = parse.Read(item2)
	be.Err(t, err, nil)
	ok = mock.Sum(t, 2, sum)
	be.Equal(t, true, ok)
}

func Test_SetBucket(t *testing.T) {
	s := parse.Scanner{}
	err := s.SetBuckets("")
	be.Err(t, err, nil)
	bucket1, err := mock.Bucket(t, 1)
	be.Err(t, err, nil)
	err = s.SetBuckets(bucket1)
	be.Err(t, err, nil)
	count := len(s.Buckets)
	expected := 1
	be.Equal(t, expected, count)
	bucket2, err := mock.Bucket(t, 2)
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
	bucket1, err := mock.Bucket(t, 1)
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
	item1 := mock.Item(t, 1)
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
	item1 := mock.Item(t, 1)
	sum1 := mock.ItemSum(t, 1)
	m[database.Filepath(item1)] = database.Bucket(sum1) // 1 match
	s = parse.Print(false, false, "", &m)
	ok := strings.Contains(s, item1)
	be.True(t, ok)
	ok = strings.Contains(s, sum1)
	be.True(t, ok)
	s = parse.Print(true, false, "", &m)
	ok = strings.Contains(s, item1)
	be.True(t, ok)
	ok = strings.Contains(s, sum1)
	be.True(t, !ok)
	// exact and term are untested as they only effect ANSI color output.
}

// BenchmarkChecksum benchmarks the Checksum function performance
func BenchmarkChecksum(b *testing.B) {
	// Create test data of different sizes
	testData := []struct {
		name string
		data []byte
	}{
		{"small", []byte("test content")},
		{"medium", bytes.Repeat([]byte("test"), 1000)},
		{"large", bytes.Repeat([]byte("test"), 10000)},
	}

	for _, tc := range testData {
		b.Run(tc.name, func(b *testing.B) {
			// Create temporary file for each benchmark iteration
			for i := 0; i < b.N; i++ {
				file, err := os.CreateTemp("", "benchmark-*.tmp")
				if err != nil {
					b.Fatalf("Failed to create temp file: %v", err)
				}
				
				// Write test data
				if _, err := file.Write(tc.data); err != nil {
					b.Fatalf("Failed to write to temp file: %v", err)
				}
				file.Close()
				
				// Benchmark checksum calculation
				_, err = parse.Read(file.Name())
				if err != nil {
					b.Fatalf("Read failed: %v", err)
				}
				
				// Clean up
				os.Remove(file.Name())
			}
		})
	}
}

// BenchmarkMarker benchmarks the Marker function performance
func BenchmarkMarker(b *testing.B) {
	testCases := []struct {
		name     string
		filepath string
		term     string
		exact    bool
	}{
		{"simple", "file.txt", "", false},
		{"with term", "path/to/file.txt", "file.txt", false},
		{"with exact", "path/to/file.txt", "file.txt", true},
		{"long path", "very/long/path/to/file.txt", "file.txt", false},
		{"unicode", "文件.txt", "", false},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = parse.Marker(database.Filepath(tc.filepath), tc.term, tc.exact)
			}
		})
	}
}

// BenchmarkPrint benchmarks the Print function performance
func BenchmarkPrint(b *testing.B) {
	// Create test matches
	matches := make(database.Matches)
	for i := 0; i < 10; i++ {
		filepath := database.Filepath(fmt.Sprintf("file%d.txt", i))
		bucket := database.Bucket(fmt.Sprintf("bucket%d", i%3))
		matches[filepath] = bucket
	}

	testCases := []struct {
		name     string
		exact    bool
		term     string
		matches  database.Matches
	}{
		{"small", false, "", database.Matches{database.Filepath("file.txt"): database.Bucket("bucket1")}},
		{"medium", false, "", matches},
		{"large", false, "", createLargeMatches()},
		{"with term", true, "file", matches},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = parse.Print(tc.exact, false, tc.term, &tc.matches)
			}
		})
	}
}

// createLargeMatches creates a large set of matches for benchmarking
func createLargeMatches() database.Matches {
	matches := make(database.Matches)
	for i := 0; i < 1000; i++ {
		filepath := database.Filepath(fmt.Sprintf("very/long/path/to/file%d.txt", i))
		bucket := database.Bucket(fmt.Sprintf("bucket%d", i%10))
		matches[filepath] = bucket
	}
	return matches
}
