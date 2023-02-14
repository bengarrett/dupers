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

func TestConfig_Print(t *testing.T) {
	b1, _ := filepath.Abs(file1)
	b2, _ := filepath.Abs(file2)
	c := dupe.Config{}
	c.Sources = []string{b1, b2}
	sum, _ := parse.Read(b1)
	c.Compare = make(parse.Checksums)
	c.Compare[sum] = file1

	s, err := c.Print()
	if err != nil {
		t.Error(err)
	}
	if s == "" {
		t.Errorf("Config.Print() should have returned a result.")
	}
}

func TestConfig_Remove(t *testing.T) {
	color.Enable = false
	c := dupe.Config{Test: true}
	r, err := c.Remove()
	if err != nil {
		t.Error(err)
	}
	if strings.TrimSpace(r) != "No duplicate files to remove." {
		t.Errorf("Config.Remove() should have returned a nothing to remove message, not %q.", r)
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
	s, err := c.Remove()
	if err != nil {
		t.Error(err)
	}
	if strings.TrimSpace(s) != want {
		t.Errorf("Config.Remove() returned an unexpected reply: %s, want %s", s, want)
	}
}

func TestConfig_Clean(t *testing.T) {
	c := dupe.Config{Test: true}
	if err := c.Clean(); err != nil {
		t.Error(err)
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
	if err := c.SetSource(filepath.Dir(rmDst)); err != nil {
		t.Error(err)
	}
	dir := filepath.Join(c.GetSource(), "empty directory placeholder")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Error(err)
	}
	// clean
	if err := c.Clean(); err != nil {
		t.Error(err)
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
	bucket1, err := mock.Bucket(1)
	if err != nil {
		t.Error(err)
	}
	c := dupe.Config{Test: true, Debug: true}
	db, err := mock.Database()
	if err != nil {
		t.Error(err)
	}
	defer db.Close()
	if err := c.SetBuckets(bucket1); err != nil {
		t.Error(err)
	}
	if err := c.WalkDirs(db); err != nil {
		t.Error(err)
	}
}

func TestConfig_WalkDir(t *testing.T) {
	bucket1, err := mock.Bucket(1)
	if err != nil {
		t.Error(err)
	}
	item1, err := mock.Item(1)
	if err != nil {
		t.Error(err)
	}
	c := dupe.Config{Test: true, Debug: true}
	db, err := mock.Database()
	if err != nil {
		t.Error(err)
	}
	defer db.Close()
	if err := c.WalkDir(db, ""); err == nil {
		t.Errorf("Config.WalkDir() should return an error with an empty Config.")
	}
	err = c.WalkDir(db, parse.Bucket(item1))
	if err != nil {
		t.Errorf("Config.WalkDir(%s) should skip files.", item1)
	}
	err = c.WalkDir(db, parse.Bucket(bucket1))
	if err != nil {
		t.Errorf("Config.WalkDir(%s) returned the error: %v", bucket1, err)
	}
}

func TestConfig_WalkSource(t *testing.T) {
	bucket2, err := mock.Bucket(2)
	if err != nil {
		t.Error(err)
	}
	c := dupe.Config{}
	if err := c.WalkSource(); err == nil {
		t.Errorf("Config.WalkSource() should return an error with an empty Config.")
	}
	if err := c.SetSource(bucket2); err != nil {
		t.Errorf("Config.SetSource() returned an error: %v", err)
	}
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
	if err := c.SetSource(abs); err != nil {
		t.Error(err)
	}
	c.Sources = append(c.Sources, srcs)
	s, err := c.Removes(false)
	if err != nil {
		t.Error(err)
	}
	fmt.Fprintln(os.Stdout, s)
}

func TestChecksum(t *testing.T) {
	bucket1, err := mock.Bucket(1)
	if err != nil {
		t.Error(err)
	}
	c := dupe.Config{Test: true, Quiet: false, Debug: true}
	db, err := mock.Database()
	if err != nil {
		t.Error(err)
	}
	defer db.Close()
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
		{"invalid path", args{"abcde", bucket1}, true},
		{"okay", args{file, bucket1}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := c.Checksum(db, tt.args.name, tt.args.bucket); (err != nil) != tt.wantErr {
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
		fmt.Fprintln(os.Stdout, path)
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
	bucket1, err := mock.Bucket(1)
	if err != nil {
		t.Error(err)
	}
	c := dupe.Config{Test: true, Quiet: false, Debug: true}
	db, err := mock.Database()
	if err != nil {
		t.Error(err)
	}
	defer db.Close()
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
		{"okay", args{bucket1}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := c.WalkArchiver(db, parse.Bucket(tt.args.name)); (err != nil) != tt.wantErr {
				t.Errorf("Config.WalkArchiver() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
