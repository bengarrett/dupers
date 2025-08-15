// © Ben Garrett https://github.com/bengarrett/dupers
package dupe_test

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bengarrett/dupers/internal/mock"
	"github.com/bengarrett/dupers/pkg/database"
	"github.com/bengarrett/dupers/pkg/dupe"
	"github.com/bengarrett/dupers/pkg/dupe/parse"
	"github.com/gookit/color"
	"github.com/nalgeon/be"
)

func TestConfig_Print(t *testing.T) {
	item1 := mock.Item(t, 1)
	item2 := mock.Item(t, 2)
	c := dupe.Config{}
	c.Sources = []string{item1, item2}
	sum, err := parse.Read(item1)
	be.Err(t, err, nil)
	c.Compare = make(parse.Checksums)
	c.Compare[sum] = item1
	s, err := c.Print()
	be.Err(t, err, nil)
	ok := strings.Contains(s, "No duplicate files found")
	be.True(t, ok)
}

func copyfile(t *testing.T, i int) (string, string) {
	item := mock.Item(t, i)
	tmpdir := t.TempDir()
	dest := filepath.Join(tmpdir, "configremovefile")
	b, err := database.CopyFile(item, dest)
	be.Err(t, err, nil)
	const written = int64(20)
	be.Equal(t, b, written)
	return item, dest
}

func TestConfig_Remove(t *testing.T) {
	color.Enable = false
	c := dupe.Config{Test: true}
	// remove nothing
	s, err := c.Remove()
	be.Err(t, err, nil)
	ok := strings.Contains(s, "No duplicate files to remove")
	be.True(t, ok)
	// copy file
	_, dest := copyfile(t, 1)
	defer os.Remove(dest)
	// setup mock sources
	c.Sources = append(c.Sources, dest)
	sum, err := parse.Read(dest)
	be.Err(t, err, nil)
	c.Compare = make(parse.Checksums)
	c.Compare[sum] = dest
	// test remove
	s, err = c.Remove()
	be.Err(t, err, nil)
	ok = strings.Contains(s, "removed:")
	be.True(t, ok)
}

func TestConfig_Clean(t *testing.T) {
	c := dupe.Config{Test: true}
	var b bytes.Buffer
	err := c.Clean(&b)
	be.Err(t, err, nil)
	// copy file
	_, dest := copyfile(t, 1)
	defer os.Remove(dest)
	// make empty dir
	tmp := t.TempDir()
	be.Err(t, err, nil)
	c.SetSource(tmp)
	err = os.MkdirAll(filepath.Join(tmp, "config-clean-empty-dir"), mock.PrivateDir)
	be.Err(t, err, nil)
	err = c.Clean(&b)
	be.Err(t, err, nil)
}

func TestConfig_Status(t *testing.T) {
	c := dupe.Config{Test: true}
	const testVal = 321
	c.Files = testVal
	s := c.Status()
	ok := strings.Contains(s, fmt.Sprintf("Scanned %d files", testVal))
	be.True(t, ok)
}

func TestConfig_WalkDirs(t *testing.T) {
	c := dupe.Config{Test: true, Debug: true}
	bucket1, err := mock.Bucket(t, 1)
	be.Err(t, err, nil)
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)
	c.SetBuckets(bucket1)
	be.Err(t, err, nil)
	err = c.WalkDirs(db)
	be.Err(t, err, nil)
}

func TestConfig_WalkDir(t *testing.T) {
	c := dupe.Config{Test: true, Debug: false}
	bucket1, err := mock.Bucket(t, 1)
	be.Err(t, err, nil)
	item1 := mock.Item(t, 1)
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)
	err = c.WalkDir(db, "")
	be.Err(t, err)
	err = c.WalkDir(db, parse.Bucket(item1))
	be.Err(t, err, nil)
	err = c.WalkDir(db, parse.Bucket(bucket1))
	be.Err(t, err, nil)
}

func TestConfig_WalkSource(t *testing.T) {
	c := dupe.Config{}
	bucket2, err := mock.Bucket(t, 2)
	be.Err(t, err, nil)
	err = c.WalkSource()
	be.Err(t, err)
	err = c.SetSource(bucket2)
	be.Err(t, err, nil)
	err = c.WalkSource()
	be.Err(t, err, nil)
}

func TestPrintWalk(t *testing.T) {
	c := dupe.Config{Test: false, Quiet: false, Debug: false}
	s := dupe.PrintWalk(false, &c)
	be.Equal(t, s, "")
	c.Files = 15
	s = dupe.PrintWalk(false, &c)
	ok := strings.Contains(s, "Scanning 15 files")
	be.True(t, ok)
	const lookup = true
	s = dupe.PrintWalk(lookup, &c)
	ok = strings.Contains(s, "Looking up 15 items")
	be.True(t, ok)
	c.Quiet = true
	s = dupe.PrintWalk(lookup, &c)
	be.Equal(t, s, "")
}

func TestRemovers(t *testing.T) {
	tmpDir := t.TempDir()
	count := mock.RemoveTmp(t, tmpDir)
	be.Equal(t, count, 0)
	bucket1, err := mock.Bucket(t, 1)
	be.Err(t, err, nil)
	be.True(t, bucket1 != "")
	path := mock.MirrorData(t)
	c := dupe.Config{Test: true, Quiet: false, Debug: false}
	err = c.SetSource(path)
	be.Err(t, err, nil)
	i := mock.SensenTmp(t, path)
	be.Equal(t, int64(20), i)
	paths, err := c.Removes()
	be.Err(t, err, nil)
	be.Equal(t, len(paths), 0)
	removed := mock.RemoveTmp(t, path)
	const expected = 33
	be.Equal(t, removed, expected)
}

func TestChecksum(t *testing.T) {
	c := dupe.Config{Test: true, Quiet: false, Debug: false}
	bucket1, err := mock.Bucket(t, 1)
	be.Err(t, err, nil)
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)
	item := mock.Item(t, 1)
	be.True(t, item != "")
	err = c.Checksum(nil, "", "")
	be.Err(t, err)
	err = c.Checksum(db, "", "")
	be.Err(t, err)
	err = c.Checksum(db, "", bucket1)
	be.Err(t, err)
	err = c.Checksum(db, "qwertyuiop", bucket1)
	be.Err(t, err)
	err = c.Checksum(db, item, bucket1)
	be.Err(t, err, nil)
}

func TestConfig_WalkArchiver(t *testing.T) {
	c := dupe.Config{Test: true, Quiet: false, Debug: false}
	bucket1, err := mock.Bucket(t, 1)
	be.Err(t, err, nil)
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)
	err = c.WalkArchiver(nil, "")
	be.Err(t, err)
	err = c.WalkArchiver(db, "qwertyuiop")
	be.Err(t, err)
	err = c.WalkArchiver(db, parse.Bucket(bucket1))
	be.Err(t, err, nil)
}

func TestConfig_Writer(t *testing.T) {
	const s = "test config dwrite"
	var c dupe.Config
	var w bytes.Buffer
	c.Writer(&w, s)
	be.Equal(t, w.String(), "")
	c.Debug = true
	c.Writer(&w, s)
	ok := strings.Contains(w.String(), s)
	be.True(t, ok)
}

func TestConfig_StatSource(t *testing.T) {
	var c dupe.Config
	_, files, buckets, err := c.StatSource()
	be.Err(t, err)
	be.Equal(t, files, 0)
	be.Equal(t, buckets, 0)
	bucket1, err := mock.Bucket(t, 1)
	be.Err(t, err, nil)
	err = c.SetSource(bucket1)
	be.Err(t, err, nil)
	err = c.SetBuckets(bucket1)
	be.Err(t, err, nil)
	_, files, buckets, err = c.StatSource()
	be.Err(t, err, nil)
	const filesCount = 24
	be.Equal(t, files, filesCount)
	be.Equal(t, buckets, filesCount)
}

func TestMatch(t *testing.T) {
	s := dupe.Match("", "")
	be.Equal(t, s, "")
	const item = "some-pretend-file"
	tmpDir := t.TempDir()
	s = dupe.Match(tmpDir, item)
	ok := strings.Contains(s, item)
	be.True(t, ok)
	item1 := mock.Item(t, 1)
	s = dupe.Match(tmpDir, item1)
	ok = strings.Contains(s, item1)
	be.True(t, ok)
}

func TestSkipDir(t *testing.T) {
	tmpDir := t.TempDir()
	info, err := os.Stat(tmpDir)
	be.Err(t, err, nil)
	dir := fs.FileInfoToDirEntry(info)
	err = dupe.SkipFS(false, false, false, dir)
	be.Err(t, err, nil)
	skipDirs := []string{"node_modules", ".hidden", "__macosx"}
	for _, elem := range skipDirs {
		name := filepath.Join(tmpDir, elem)
		err = os.MkdirAll(name, mock.PrivateDir)
		be.Err(t, err, nil)
		defer os.Remove(name)
		info, err = os.Stat(name)
		be.Err(t, err, nil)
		dir = fs.FileInfoToDirEntry(info)
		err = dupe.SkipFS(false, false, false, dir)
		be.Err(t, err)
	}
}

func TestSkipFile(t *testing.T) {
	skipFiles := []string{".DS_STORE", "pagefile.sys", "thumbs.db"}
	for _, name := range skipFiles {
		b := dupe.SkipFile(name)
		be.True(t, b)
	}
}
