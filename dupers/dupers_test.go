// © Ben Garrett https://github.com/bengarrett/dupers

package dupers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bengarrett/dupers/database"
)

const (
	bucket1 = "../test/bucket1"
	bucket2 = "../test/bucket2"
	file1   = "../test/bucket1/0vlLaUEvzAWP"
	file2   = "../test/bucket1/GwejJkMzs3yP"
	rmSrc   = "../test/bucket1/mPzd5cu0Gv5j"
	rmDst   = "../test/tmp/mPzd5cu0Gv5j"
	// checksums created from sha256sum <filename>.
	hash0 = "0000000000000000000000000000000000000000000000000000000000000000"
	hash1 = "1a1d76a3187ccee147e6c807277273afbad5d2680f5eadf1012310743e148f22"
	hash2 = "4acc274c2e6dc2241029c735758f672b3dc1109ab76a91fe29aeb2bac6949eb7"
)

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
	i := internal{}
	i.SetToCheck(bucket1)
	t.Run("test set", func(t *testing.T) {
		if s := i.source; s == "" {
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
		{"empty", f, args{}, false, 0, 0},
		{"source", f, args{source: bucket2}, false, 2, 0},
		{"okay", f, args{source: bucket2, buckets: []Bucket{bucket1}}, true, 2, 1},
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
	c.compare[checksum(sum)] = file1

	if s := c.Print(); s == "" {
		t.Errorf("Config.Print() should have returned a result.")
	}
}

func TestConfig_Remove(t *testing.T) {
	c := Config{}
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
	c := Config{}
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
	if r := strings.TrimSpace(c.Clean()); !strings.Contains(r, "Removed 2 empty directories in:") {
		t.Errorf("Config.Clean() should have returned a remove notice, not %v.", r)
	}
}
