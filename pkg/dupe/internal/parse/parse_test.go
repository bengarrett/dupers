// © Ben Garrett https://github.com/bengarrett/dupers
package parse_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/bengarrett/dupers/internal/mock"
	"github.com/bengarrett/dupers/pkg/database"
	"github.com/bengarrett/dupers/pkg/dupe/internal/parse"
	"github.com/gookit/color"
)

const (
	bucket1 = "../../../test/bucket1/"
	file1   = "../../../test/bucket1/0vlLaUEvzAWP"
	file2   = "../../../test/bucket1/GwejJkMzs3yP"
	sensen  = "../../../test/sensen/"
	// checksums created from sha256sum <filename>.
	hash0 = "0000000000000000000000000000000000000000000000000000000000000000"
	hash1 = "1a1d76a3187ccee147e6c807277273afbad5d2680f5eadf1012310743e148f22"
	hash2 = "4acc274c2e6dc2241029c735758f672b3dc1109ab76a91fe29aeb2bac6949eb7"
)

func ExamplePrint() {
	color.Enable = false
	matches := database.Matches{}
	matches[database.Filepath(file1)] = database.Bucket(bucket1)
	s := parse.Print(true, true, "", &matches)
	fmt.Print(s)
	// Output: ../../../test/bucket1/0vlLaUEvzAWP
}

func TestSetBuckets(t *testing.T) {
	database.TestMode = true
	db, err := mock.Database()
	if err != nil {
		t.Error(err)
	}
	defer db.Close()
	var p parse.Scanner
	if err := p.SetAllBuckets(db); err != nil {
		t.Error(err)
		return
	}
	const expected = 2
	if l := len(p.All()); l != expected {
		t.Errorf("Expected %d, got %d", expected, l)
	}
}

func TestTimer(t *testing.T) {
	p := parse.Scanner{}
	p.SetTimer()
	if z := p.Timer(); z == 0 {
		t.Error("timer should not be zero")
	}
}

func TestParser_SetCompares(t *testing.T) {
	bucket1, err := mock.Bucket(1)
	if err != nil {
		t.Error(err)
	}
	bucket2, err := mock.Bucket(2)
	if err != nil {
		t.Error(err)
	}
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
		want    int
		wantErr bool
	}{
		{"empty", args{}, 0, true},
		{"mock1", args{bucket1}, 4, false},
		{"mock2", args{bucket2}, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := parse.Scanner{}
			got, err := p.SetCompares(db, parse.Bucket(tt.args.name))
			if (err != nil) != tt.wantErr {
				t.Errorf("Scanner.SetCompares() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Scanner.SetCompares() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContains(t *testing.T) {
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
			if got := parse.Contains(tt.args.find, tt.args.s...); got != tt.want {
				t.Errorf("contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExecutable(t *testing.T) {
	d, err := filepath.Abs(bucket1)
	if err != nil {
		t.Error(err)
	}
	tests := []struct {
		name    string
		root    string
		want    bool
		wantErr bool
	}{
		{"test dir", d, false, false},
		{"test a file", file1, false, false},
		{"test dir with an exe", sensen, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := parse.Executable(tt.root)
			if got != tt.want {
				t.Errorf("Executable() = %v, want %v", got, tt.want)
			}
			if (gotErr != nil) != tt.wantErr {
				t.Errorf("Executable() error = %v, want %v", (gotErr != nil), tt.wantErr)
			}
		})
	}
}

func TestRead(t *testing.T) {
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
			gotHash, err := parse.Read(tt.path)
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

func Test_SetBucket(t *testing.T) {
	i := parse.Scanner{}
	if err := i.SetBuckets(bucket1); err != nil {
		t.Error(err)
		return
	}
	t.Run("test set", func(t *testing.T) {
		if l := len(i.Buckets); l != 1 {
			t.Errorf("SetBuckets() got = %v, want %v", l, 1)
		}
	})
	t.Run("print", func(t *testing.T) {
		if s := i.BucketS(); s != bucket1 {
			t.Errorf("SetBuckets() got = %v, want %v", s, bucket1)
		}
	})
}

func Test_SetSource(t *testing.T) {
	c := parse.Scanner{}
	if err := c.SetSource(bucket1); err != nil {
		t.Errorf("SetSource(%v) returned the error: %v", bucket1, err)
	}
	t.Run("test set", func(t *testing.T) {
		if s := c.GetSource(); s == "" {
			t.Errorf("SetSource() got = %v, want the absolute path of: %v", s, bucket1)
		}
	})
}

func TestMarker(t *testing.T) {
	type args struct {
		file  string
		term  string
		exact bool
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"empty", args{}, ""},
		{"no term", args{file1, "", true}, file1},
		{"good", args{file1, "awp", false}, "../../../test/bucket1/0vlLaUEvzAWP"},
		{"invalid case", args{file1, "awp", true}, file1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// turn off color to force a different output
			color.Enable = false
			if got := parse.Marker(database.Filepath(tt.args.file), tt.args.term, tt.args.exact); got != tt.want {
				t.Errorf("Marker() = %q, want %q", got, tt.want)
			}
		})
	}
}