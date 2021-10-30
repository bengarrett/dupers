// © Ben Garrett https://github.com/bengarrett/dupers

package dupe

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/bengarrett/dupers/database"
	"github.com/bengarrett/dupers/mock"
	"github.com/bengarrett/dupers/out"
	"github.com/gookit/color"
)

const (
	bucket0 = "../test/tmp"
	bucket1 = "../test/bucket1"
	bucket2 = "../test/bucket2"
	bucket3 = "../test/sensen"
	file1   = "../test/bucket1/0vlLaUEvzAWP"
	file2   = "../test/bucket1/GwejJkMzs3yP"
	rmSrc   = "../test/bucket1/mPzd5cu0Gv5j"
	rmDst   = "../test/tmp/mPzd5cu0Gv5j"
	// checksums created from sha256sum <filename>.
	hash0 = "0000000000000000000000000000000000000000000000000000000000000000"
	hash1 = "1a1d76a3187ccee147e6c807277273afbad5d2680f5eadf1012310743e148f22"
	hash2 = "4acc274c2e6dc2241029c735758f672b3dc1109ab76a91fe29aeb2bac6949eb7"
)

func init() { // nolint:gochecknoinits
	color.Enable = false
}

func ExamplePrint() {
	matches := database.Matches{}
	matches[database.Filepath(file1)] = database.Bucket(bucket1)
	s := Print(true, &matches)
	fmt.Print(s)
	// Output: ../test/bucket1/0vlLaUEvzAWP
}

func Test_contains(t *testing.T) {

	type args struct {
		s    []string
		find string
	}
	str := []string{"abc", "def", "ghijkl"}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"empty", args{nil, ""}, false},
		{"no find", args{str, ""}, false},
		{"find", args{str, "def"}, true},
		{"partial", args{str, "de"}, false},
		{"find upper", args{str, "DEF"}, false},
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			if got := contains(tt.args.find, tt.args.s...); got != tt.want {
				t.Errorf("contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_containsBin(t *testing.T) {

	d, err := filepath.Abs(bucket1)
	if err != nil {
		t.Error(err)
	}
	tests := []struct {
		name string
		root string
		want bool
	}{
		{"test dir", d, false},
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			if got := containsBin(tt.root); got != tt.want {
				t.Errorf("containsBin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_read(t *testing.T) {

	tests := []struct {
		name     string
		path     string
		wantHash string
		wantErr  bool
	}{
		{"empty", "", hash0, true},
		{"file1", file1, hash1, false},
		{"file2", file2, hash2, false},
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			gotHash, err := read(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("read() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			h := fmt.Sprintf("%x", gotHash)
			if h != tt.wantHash {
				t.Errorf("read() got = %v, want %v", h, tt.wantHash)
			}
		})
	}
}

func Test_SetBuckets(t *testing.T) {

	const test = "test"
	i := internal{}
	i.SetBuckets(test)
	t.Run("test set", func(t *testing.T) {

		if l := len(i.buckets); l != 1 {
			t.Errorf("SetBuckets() got = %v, want %v", l, 1)
		}
	})
	t.Run("print", func(t *testing.T) {

		if s := i.PrintBuckets(); s != test {
			t.Errorf("SetBuckets() got = %v, want %v", s, test)
		}
	})
}

func Test_SetToCheck(t *testing.T) {

	c := Config{}
	c.SetToCheck(bucket1)
	t.Run("test set", func(t *testing.T) {

		if s := c.source; s == "" {
			t.Errorf("SetToCheck() got = %v, want the absolute path of: %v", s, bucket1)
		}
	})
}

func TestConfig_CheckPaths(t *testing.T) {

	type fields struct {
		Debug    bool
		Quiet    bool
		Test     bool
		internal internal
	}
	type args struct {
		source  string
		buckets []Bucket
	}
	f := fields{Test: true}
	tests := []struct {
		name          string
		fields        fields
		args          args
		wantOk        bool
		wantCheckCnt  int
		wantBucketCnt int
	}{
		{"empty", f, args{}, true, 0, 0},
		{"source", f, args{source: bucket2}, false, 3, 0},
		{"okay", f, args{source: bucket2, buckets: []Bucket{bucket1}}, true, 3, 1},
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			c := &Config{
				Debug:    tt.fields.Debug,
				Quiet:    tt.fields.Quiet,
				Test:     tt.fields.Test,
				internal: tt.fields.internal,
			}
			c.source = tt.args.source
			c.buckets = tt.args.buckets
			gotOk, gotCheckCnt, gotBucketCnt := c.CheckPaths()
			if gotOk != tt.wantOk {
				t.Errorf("Config.CheckPaths() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
			if gotCheckCnt != tt.wantCheckCnt {
				t.Errorf("Config.CheckPaths() gotCheckCnt = %v, want %v", gotCheckCnt, tt.wantCheckCnt)
			}
			if gotBucketCnt != tt.wantBucketCnt {
				t.Errorf("Config.CheckPaths() gotBucketCnt = %v, want %v", gotBucketCnt, tt.wantBucketCnt)
			}
		})
	}
}

func TestConfig_Print(t *testing.T) {

	b1, _ := filepath.Abs(file1)
	b2, _ := filepath.Abs(file2)
	c := Config{}
	c.sources = []string{b1, b2}
	sum, _ := read(b1)
	c.compare = make(checksums)
	c.compare[sum] = file1

	if s := c.Print(); s == "" {
		t.Errorf("Config.Print() should have returned a result.")
	}
}

func TestConfig_Remove(t *testing.T) {

	c := Config{Test: true}
	if r := strings.TrimSpace(c.Remove()); r != "No duplicate files to remove." {
		t.Errorf("Config.Remove() should have returned a nothing to remove message, not %v.", r)
	}
	// copy file
	const written = 20
	i, err := database.CopyFile(rmSrc, rmDst)
	if err != nil {
		t.Error(err)
	}

	defer os.Remove(rmDst)
	if i != written {
		t.Errorf("CopyFile should have written %d bytes, but wrote %d", written, i)
	}
	// setup mock databases
	c.sources = append(c.sources, rmDst)
	sum, err := read(rmDst)
	if err != nil {
		t.Error(err)
	}
	c.compare = make(checksums)
	c.compare[sum] = rmDst
	want := fmt.Sprintf("removed: %s", rmDst)
	if s := c.Remove(); strings.TrimSpace(s) != want {
		t.Errorf("Config.Remove() returned an unexpected reply: %s, want %s", s, want)
	}
}

func TestConfig_Clean(t *testing.T) {

	c := Config{Test: true}
	if r := strings.TrimSpace(c.Clean()); r != "" {
		t.Errorf("Config.Clean() should have returned blank, not %v.", r)
	}
	// copy file
	const written = 20
	i, err := database.CopyFile(rmSrc, rmDst)
	if err != nil {
		t.Error(err)
	}

	defer os.Remove(rmDst)
	if i != written {
		t.Errorf("CopyFile should have written %d bytes, but wrote %d", written, i)
	}
	// make empty test dir
	c.source = filepath.Dir(rmDst)
	dir := filepath.Join(c.source, "empty directory placeholder")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Error(err)
	}
	// clean
	if r := strings.TrimSpace(c.Clean()); !strings.Contains(r, "Removed ") {
		t.Errorf("Config.Clean() should have returned a remove notice, not %v.", r)
	}
}

func TestConfig_Status(t *testing.T) {

	c := Config{Test: true}
	c.files = 2
	const want = "Scanned 2 files"
	if s := strings.TrimSpace(c.Status()); !strings.Contains(s, want) {
		t.Errorf("Config.Status() should contain %s, got %s", want, s)
	}
}

func TestConfig_WalkDirs(t *testing.T) {

	var err error
	c := Config{Test: true, Debug: true}
	c.db, err = mock.Open()
	if err != nil {
		t.Error(err)
	}
	defer c.db.Close()
	c.SetBuckets(mock.Bucket1())
	c.WalkDirs()
}

func TestConfig_WalkDir(t *testing.T) {

	var err error
	c := Config{Test: true, Debug: true}
	c.db, err = mock.Open()
	if err != nil {
		t.Error(err)
	}
	defer c.db.Close()
	if err := c.WalkDir(""); err == nil {
		t.Errorf("Config.WalkDir() should return an error with an empty Config.")
	}
	f := mock.Item1()
	err = c.WalkDir(Bucket(f))
	if err != nil {
		t.Errorf("Config.WalkDir(%s) should skip files.", f)
	}
	b := mock.Bucket1()
	err = c.WalkDir(Bucket(b))
	if err != nil {
		t.Errorf("Config.WalkDir(%s) returned the error: %v", b, err)
	}
}

func TestConfig_WalkSource(t *testing.T) {

	c := Config{}
	if err := c.WalkSource(); err == nil {
		t.Errorf("Config.WalkSource() should return an error with an empty Config.")
	}
	c.SetToCheck(mock.Bucket2())
	if err := c.WalkSource(); err != nil {
		t.Errorf("Config.WalkSource() returned an error: %v", err)
	}
}

func Test_printWalk(t *testing.T) {

	c := Config{Test: false, Quiet: false, Debug: false}
	s := strings.TrimSpace(printWalk(false, &c))
	want := ""
	if runtime.GOOS != winOS {
		want = out.EraseLine + "\r"
	}
	want += "Scanning 0 files"
	if s != want {
		t.Errorf("printWalk() returned: %s, want %s", s, want)
	}
	c.files = 15
	s = strings.TrimSpace(printWalk(false, &c))
	want = ""
	if runtime.GOOS != winOS {
		want = out.EraseLine + "\r"
	}
	want += "Scanning 15 files"
	if s != want {
		t.Errorf("printWalk() returned: %s, want %s", s, want)
	}
	s = strings.TrimSpace(printWalk(true, &c))
	want = ""
	if runtime.GOOS != winOS {
		want = out.EraseLine + "\r"
	}
	want += "Looking up 15 items"
	if s != want {
		t.Errorf("printWalk() returned: %s, want %s", s, want)
	}
	c.Quiet = true
	s = strings.TrimSpace(printWalk(true, &c))
	want = ""
	if s != want {
		t.Errorf("printWalk() returned: %s, want a blank string", s)
	}
}

func TestRemoveAll(t *testing.T) {

	c := Config{Test: true, Quiet: false, Debug: true}
	if err := cleanDir(bucket0); err != nil {
		t.Error(err)
	}
	if err := mirrorDir(bucket3, bucket0); err != nil {
		t.Error(err)
	}
	abs, err := filepath.Abs(bucket0)
	if err != nil {
		t.Error(err)
	}
	srcs, err := filepath.Abs(bucket2)
	if err != nil {
		t.Error(err)
	}
	c.source = abs
	c.sources = append(c.sources, srcs)
	s := c.RemoveAll()
	fmt.Println(s)
}

func cleanDir(name string) error {
	abs, err := filepath.Abs(name)
	if err != nil {
		return err
	}
	return filepath.WalkDir(abs, func(path string, d fs.DirEntry, err error) error {
		if path == abs {
			return nil
		}
		if _, err := os.Stat(path); err != nil {
			return err
		}
		fmt.Println(path)
		if err := os.RemoveAll(path); err != nil {
			log.Println(err)
		}
		return nil
	})
}

func mirrorDir(src, dst string) error {
	const dirAllAccess fs.FileMode = 0777
	from, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	to, err := filepath.Abs(dst)
	if err != nil {
		return err
	}
	return filepath.WalkDir(from, func(path string, d fs.DirEntry, err error) error {
		if path == from {
			return nil
		}
		dest := filepath.Join(to, strings.Replace(path, from, "", 1))
		if d.IsDir() {
			if errM := os.MkdirAll(dest, dirAllAccess); errM != nil {
				log.Println(errM)
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if _, errC := database.CopyFile(path, dest); errC != nil {
			log.Println(errC)
		}
		return nil
	})
}
