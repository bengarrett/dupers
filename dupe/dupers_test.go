// © Ben Garrett https://github.com/bengarrett/dupers

package dupe_test

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
	"github.com/bengarrett/dupers/dupe"
	"github.com/bengarrett/dupers/dupe/internal/parse"
	"github.com/bengarrett/dupers/internal/mock"
	"github.com/bengarrett/dupers/internal/out"
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
)

func TestConfig_CheckPaths(t *testing.T) {
	type fields struct {
		Debug bool
		Quiet bool
		Test  bool
		parse.Parser
	}
	type args struct {
		source  string
		buckets []parse.Bucket
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
		{"okay", f, args{source: bucket2, buckets: []parse.Bucket{bucket1}}, true, 3, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &dupe.Config{
				Debug:  tt.fields.Debug,
				Quiet:  tt.fields.Quiet,
				Test:   tt.fields.Test,
				Parser: tt.fields.Parser,
			}
			c.Source = tt.args.source
			c.Buckets = tt.args.buckets
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
	c := dupe.Config{}
	c.Sources = []string{b1, b2}
	sum, _ := parse.Read(b1)
	c.Compare = make(parse.Checksums)
	c.Compare[sum] = file1

	if s := c.Print(); s == "" {
		t.Errorf("Config.Print() should have returned a result.")
	}
}

func TestConfig_Remove(t *testing.T) {
	color.Enable = false
	c := dupe.Config{Test: true}
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
	c.Sources = append(c.Sources, rmDst)
	sum, err := parse.Read(rmDst)
	if err != nil {
		t.Error(err)
	}
	c.Compare = make(parse.Checksums)
	c.Compare[sum] = rmDst
	want := fmt.Sprintf("removed: %s", rmDst)
	if s := c.Remove(); strings.TrimSpace(s) != want {
		t.Errorf("Config.Remove() returned an unexpected reply: %s, want %s", s, want)
	}
}

func TestConfig_Clean(t *testing.T) {
	c := dupe.Config{Test: true}
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
	c.Source = filepath.Dir(rmDst)
	dir := filepath.Join(c.Source, "empty directory placeholder")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Error(err)
	}
	// clean
	if r := strings.TrimSpace(c.Clean()); !strings.Contains(r, "Removed ") {
		t.Errorf("Config.Clean() should have returned a remove notice, not %v.", r)
	}
}

func TestConfig_Status(t *testing.T) {
	c := dupe.Config{Test: true}
	c.Files = 2
	const want = "Scanned 2 files"
	if s := strings.TrimSpace(c.Status()); !strings.Contains(s, want) {
		t.Errorf("Config.Status() should contain %s, got %s", want, s)
	}
}

func TestConfig_WalkDirs(t *testing.T) {
	var err error
	c := dupe.Config{Test: true, Debug: true}
	c.DB, err = mock.Open()
	if err != nil {
		t.Error(err)
	}
	defer c.DB.Close()
	c.SetBucket(mock.Bucket1())
	c.WalkDirs()
}

func TestConfig_WalkDir(t *testing.T) {
	var err error
	c := dupe.Config{Test: true, Debug: true}
	c.DB, err = mock.Open()
	if err != nil {
		t.Error(err)
	}
	defer c.DB.Close()
	if err := c.WalkDir(""); err == nil {
		t.Errorf("Config.WalkDir() should return an error with an empty Config.")
	}
	f := mock.Item1()
	err = c.WalkDir(parse.Bucket(f))
	if err != nil {
		t.Errorf("Config.WalkDir(%s) should skip files.", f)
	}
	b := mock.Bucket1()
	err = c.WalkDir(parse.Bucket(b))
	if err != nil {
		t.Errorf("Config.WalkDir(%s) returned the error: %v", b, err)
	}
}

func TestConfig_WalkSource(t *testing.T) {
	c := dupe.Config{}
	if err := c.WalkSource(); err == nil {
		t.Errorf("Config.WalkSource() should return an error with an empty Config.")
	}
	c.SetToCheck(mock.Bucket2())
	if err := c.WalkSource(); err != nil {
		t.Errorf("Config.WalkSource() returned an error: %v", err)
	}
}

func TestPrintWalk(t *testing.T) {
	c := dupe.Config{Test: false, Quiet: false, Debug: false}
	s := strings.TrimSpace(dupe.PrintWalk(false, &c))
	want := ""
	if runtime.GOOS != dupe.WinOS {
		want = out.EraseLine + "\r"
	}
	want += "Scanning 0 files"
	if s != want {
		t.Errorf("PrintWalk() returned: %s, want %s", s, want)
	}
	c.Files = 15
	s = strings.TrimSpace(dupe.PrintWalk(false, &c))
	want = ""
	if runtime.GOOS != dupe.WinOS {
		want = out.EraseLine + "\r"
	}
	want += "Scanning 15 files"
	if s != want {
		t.Errorf("PrintWalk() returned: %s, want %s", s, want)
	}
	s = strings.TrimSpace(dupe.PrintWalk(true, &c))
	want = ""
	if runtime.GOOS != dupe.WinOS {
		want = out.EraseLine + "\r"
	}
	want += "Looking up 15 items"
	if s != want {
		t.Errorf("PrintWalk() returned: %s, want %s", s, want)
	}
	c.Quiet = true
	s = strings.TrimSpace(dupe.PrintWalk(true, &c))
	want = ""
	if s != want {
		t.Errorf("PrintWalk() returned: %s, want a blank string", s)
	}
}

func TestRemoves(t *testing.T) {
	c := dupe.Config{Test: true, Quiet: false, Debug: true}
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
	c.Source = abs
	c.Sources = append(c.Sources, srcs)
	s := c.Removes()
	fmt.Println(s)
}

func TestChecksum(t *testing.T) {
	c := dupe.Config{Test: true, Quiet: false, Debug: true}
	var err error
	c.DB, err = mock.Open()
	if err != nil {
		c.DB.Close()
		t.Error(err)
		return
	}
	defer c.DB.Close()
	file, err := filepath.Abs(file1)
	if err != nil {
		t.Error(err)
		return
	}
	type args struct {
		name   string
		bucket string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"empty", args{}, true},
		{"invalid path", args{"abcde", mock.Bucket1()}, true},
		{"okay", args{file, mock.Bucket1()}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := c.Checksum(tt.args.name, tt.args.bucket); (err != nil) != tt.wantErr {
				t.Errorf("Config.Checksum() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
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
	const dirAllAccess fs.FileMode = 0o777
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

func TestConfig_WalkArchiver(t *testing.T) {
	c := dupe.Config{Test: true, Quiet: false, Debug: true}
	var err error
	c.DB, err = mock.Open()
	if err != nil {
		c.DB.Close()
		t.Error(err)
		return
	}
	defer c.DB.Close()
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"empty", args{}, true},
		{"invalid", args{"abcdef"}, true},
		{"okay", args{mock.Bucket1()}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := c.WalkArchiver(parse.Bucket(tt.args.name)); (err != nil) != tt.wantErr {
				t.Errorf("Config.WalkArchiver() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
