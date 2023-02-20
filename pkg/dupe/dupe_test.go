// © Ben Garrett https://github.com/bengarrett/dupers

package dupe_test

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/bengarrett/dupers/internal/mock"
	"github.com/bengarrett/dupers/pkg/database"
	"github.com/bengarrett/dupers/pkg/dupe"
	"github.com/bengarrett/dupers/pkg/dupe/parse"
	"github.com/gookit/color"
	"github.com/stretchr/testify/assert"
)

func TestConfig_Print(t *testing.T) {
	item1, err := mock.Item(1)
	assert.Nil(t, err)

	item2, err := mock.Item(2)
	assert.Nil(t, err)

	c := dupe.Config{}
	c.Sources = []string{item1, item2}

	sum, err := parse.Read(item1)
	assert.Nil(t, err)

	c.Compare = make(parse.Checksums)
	c.Compare[sum] = item1

	s, err := c.Print()
	assert.Nil(t, err)
	assert.Contains(t, s, "No duplicate files found")
}

func copyfile(t *testing.T, i int) (string, string) {
	item, err := mock.Item(i)
	assert.Nil(t, err)
	tmpdir, err := mock.TempDir()
	assert.Nil(t, err)
	assert.NotEqual(t, "", tmpdir)
	dest := filepath.Join(tmpdir, "configremovefile")
	b, err := database.CopyFile(item, dest)
	assert.Nil(t, err)
	const written = int64(20)
	assert.Equal(t, written, b, "copyfile didnt write the expected number of bytes")
	return item, dest
}

func TestConfig_Remove(t *testing.T) {
	color.Enable = false
	c := dupe.Config{Test: true}

	// remove nothing
	s, err := c.Remove()
	assert.Nil(t, err)
	assert.Contains(t, s, "No duplicate files to remove")

	// copy file
	_, dest := copyfile(t, 1)
	defer os.Remove(dest)

	// setup mock sources
	c.Sources = append(c.Sources, dest)
	sum, err := parse.Read(dest)
	assert.Nil(t, err)
	c.Compare = make(parse.Checksums)
	c.Compare[sum] = dest

	// test remove
	s, err = c.Remove()
	assert.Nil(t, err)
	assert.Contains(t, s, "removed:")
}

func TestConfig_Clean(t *testing.T) {
	c := dupe.Config{Test: true}
	err := c.Clean()
	assert.Nil(t, err)

	// copy file
	_, dest := copyfile(t, 1)
	defer os.Remove(dest)

	// make empty dir
	tmp, err := mock.TempDir()
	assert.Nil(t, err)
	c.SetSource(tmp)
	err = os.MkdirAll(filepath.Join(tmp, "config-clean-empty-dir"), mock.PrivateDir)
	assert.Nil(t, err)
	err = c.Clean()
	assert.Nil(t, err)
}

func TestConfig_Status(t *testing.T) {
	c := dupe.Config{Test: true}
	const testVal = 321
	c.Files = testVal
	s := c.Status()
	assert.Contains(t, s, fmt.Sprintf("Scanned %d files", testVal))
}

func TestConfig_WalkDirs(t *testing.T) {
	c := dupe.Config{Test: true, Debug: true}

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)

	db, path, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()
	defer os.Remove(path)

	c.SetBuckets(bucket1)
	assert.Nil(t, err)

	err = c.WalkDirs(db)
	assert.Nil(t, err)
}

func TestConfig_WalkDir(t *testing.T) {
	c := dupe.Config{Test: true, Debug: false}

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)

	item1, err := mock.Item(1)
	assert.Nil(t, err)

	db, path, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()
	defer os.Remove(path)

	err = c.WalkDir(db, "")
	assert.NotNil(t, err, "WalkDir should return an error with the empty config")

	err = c.WalkDir(db, parse.Bucket(item1))
	assert.Nil(t, err, "WalkDir should ignore and skip any files")

	err = c.WalkDir(db, parse.Bucket(bucket1))
	assert.Nil(t, err)
}

func TestConfig_WalkSource(t *testing.T) {
	c := dupe.Config{}

	bucket2, err := mock.Bucket(2)
	assert.Nil(t, err)

	err = c.WalkSource()
	assert.NotNil(t, err, "WalkSource should return an error with the empty config")

	err = c.SetSource(bucket2)
	assert.Nil(t, err)

	err = c.WalkSource()
	assert.Nil(t, err)
}

func TestPrintWalk(t *testing.T) {
	c := dupe.Config{Test: false, Quiet: false, Debug: false}

	s := dupe.PrintWalk(false, &c)
	assert.Equal(t, "", s, "PrintWalk should return an empty string with the empty config")

	c.Files = 15
	s = dupe.PrintWalk(false, &c)
	assert.Contains(t, s, "Scanning 15 files")

	const lookup = true
	s = dupe.PrintWalk(lookup, &c)
	assert.Contains(t, s, "Looking up 15 items")

	c.Quiet = true
	s = dupe.PrintWalk(lookup, &c)
	assert.Equal(t, "", s, "Quiet mode should return an empty string")
}

func TestRemoves(t *testing.T) {
	c := dupe.Config{Test: true, Quiet: false, Debug: false}

	tmpDir, err := mock.TempDir()
	assert.Nil(t, err)

	_, err = mock.RemoveTmp(tmpDir)
	assert.Nil(t, err)

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	assert.NotNil(t, bucket1)

	path, err := mock.MirrorTmp(bucket1)
	assert.Nil(t, err)

	err = c.SetSource(path)
	assert.Nil(t, err)

	i, err := mock.SensenTmp(path)
	assert.Nil(t, err)
	assert.Equal(t, int64(20), i)

	paths, err := c.Removes()
	assert.Nil(t, err)
	assert.Len(t, paths, 0, "removes should not return any invalid paths")

	removed := 0
	removed, err = mock.RemoveTmp(path)
	assert.Nil(t, err)
	assert.Equal(t, 25, removed)
}

func TestChecksum(t *testing.T) {
	c := dupe.Config{Test: true, Quiet: false, Debug: false}

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)

	db, path, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()
	defer os.Remove(path)

	item, err := mock.Item(1)
	assert.Nil(t, err)
	assert.NotEqual(t, "", item)

	err = c.Checksum(nil, "", "")
	assert.NotNil(t, err)

	err = c.Checksum(db, "", "")
	assert.NotNil(t, err)

	err = c.Checksum(db, "", bucket1)
	assert.NotNil(t, err)

	err = c.Checksum(db, "qwertyuiop", bucket1)
	assert.NotNil(t, err)

	err = c.Checksum(db, item, bucket1)
	assert.Nil(t, err)
}

func TestConfig_WalkArchiver(t *testing.T) {
	c := dupe.Config{Test: true, Quiet: false, Debug: false}

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)

	db, path, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()
	defer os.Remove(path)

	err = c.WalkArchiver(nil, "")
	assert.NotNil(t, err)

	err = c.WalkArchiver(db, "qwertyuiop")
	assert.NotNil(t, err)

	err = c.WalkArchiver(db, parse.Bucket(bucket1))
	assert.Nil(t, err)
}

func TestConfig_Writer(t *testing.T) {
	const s = "test config dwrite"
	var c dupe.Config
	var w bytes.Buffer
	c.Writer(&w, s)
	assert.Equal(t, "", w.String())
	c.Debug = true
	c.Writer(&w, s)
	assert.Contains(t, w.String(), s)
}

func TestConfig_CheckPaths(t *testing.T) {
	var c dupe.Config
	files, buckets, err := c.CheckPaths()
	assert.NotNil(t, err)
	assert.Equal(t, 0, files, "unexpected file count")
	assert.Equal(t, 0, buckets, "unexpected bucket count")

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	err = c.SetSource(bucket1)
	assert.Nil(t, err)
	err = c.SetBuckets(bucket1)
	assert.Nil(t, err)
	files, buckets, err = c.CheckPaths()
	assert.Nil(t, err)
	assert.Equal(t, 24, files, "unexpected file count")
	assert.Equal(t, 12, buckets, "unexpected bucket count")
}

func TestMatch(t *testing.T) {
	s := dupe.Match("", "")
	assert.Equal(t, "", s)

	const item = "some-pretend-file"
	tmpDir, err := mock.TempDir()
	assert.Nil(t, err)
	s = dupe.Match(tmpDir, item)
	assert.Contains(t, s, item)

	item1, err := mock.Item(1)
	assert.Nil(t, err)
	s = dupe.Match(tmpDir, item1)
	assert.Contains(t, s, item1)
}

func TestSkipDir(t *testing.T) {
	tmpDir, err := mock.TempDir()
	assert.Nil(t, err)

	info, err := os.Stat(tmpDir)
	assert.Nil(t, err)
	dir := fs.FileInfoToDirEntry(info)
	err = dupe.SkipDir(dir)
	assert.Nil(t, err)

	skipDirs := []string{"node_modules", ".hidden", "__macosx"}
	for _, elem := range skipDirs {
		name := filepath.Join(tmpDir, elem)
		err = os.MkdirAll(name, mock.PrivateDir)
		assert.Nil(t, err)
		defer os.Remove(name)
		info, err = os.Stat(name)
		assert.Nil(t, err)
		dir = fs.FileInfoToDirEntry(info)
		err = dupe.SkipDir(dir)
		assert.NotNil(t, err)
	}
}

func TestSkipFile(t *testing.T) {
	skipFiles := []string{".DS_STORE", "pagefile.sys", "thumbs.db"}
	for _, name := range skipFiles {
		b := dupe.SkipFile(name)
		assert.Equal(t, true, b)
	}
}
