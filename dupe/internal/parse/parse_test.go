// © Ben Garrett https://github.com/bengarrett/dupers
package parse_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/bengarrett/dupers/database"
	"github.com/bengarrett/dupers/dupe/internal/parse"
	"github.com/bengarrett/dupers/internal/mock"
	"github.com/gookit/color"
)

const (
	bucket1 = "../../../test/bucket1"
	file1   = "../../../test/bucket1/0vlLaUEvzAWP"
	file2   = "../../../test/bucket1/GwejJkMzs3yP"
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

func TestParser_OpenRead(t *testing.T) {
	p := parse.Parser{}
	p.OpenRead()
	defer p.DB.Close()
	if p.DB == nil {
		t.Error("DB should not be nil")
	}
}

func TestParser_OpenWrite(t *testing.T) {
	p := parse.Parser{}
	p.OpenWrite()
	defer p.DB.Close()
	if p.DB == nil {
		t.Error("DB should not be nil")
	}
}

func TestSetBuckets(t *testing.T) {
	database.TestMode = true
	err := mock.TestOpen()
	if err != nil {
		t.Error(err)
	}
	db, err := mock.TestDB()
	if err != nil {
		t.Error(err)
	}
	defer db.Close()
	p := parse.Parser{
		DB: db,
	}
	if err := p.SetAllBuckets(); err != nil {
		t.Error(err)
		return
	}
	const expected = 1
	if l := len(p.All()); l != expected {
		t.Errorf("Expected %d, got %d", expected, l)
	}
}

func TestTimer(t *testing.T) {
	p := parse.Parser{}
	p.SetTimer()
	if z := p.Timer(); z == 0 {
		t.Error("timer should not be zero")
	}
}

func TestParser_SetCompares(t *testing.T) {
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
		// {"mock1", args{mock.Bucket1()}, 26, false},
		// {"mock2", args{mock.Bucket2()}, 4, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := parse.Parser{}
			got, err := p.SetCompares(parse.Bucket(tt.args.name))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parser.SetCompares() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Parser.SetCompares() = %v, want %v", got, tt.want)
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
		name string
		root string
		want bool
	}{
		{"test dir", d, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parse.Executable(tt.root); got != tt.want {
				t.Errorf("Executable() = %v, want %v", got, tt.want)
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
	i := parse.Parser{}
	if err := i.SetBucket(bucket1); err != nil {
		t.Error(err)
		return
	}
	t.Run("test set", func(t *testing.T) {
		if l := len(i.Buckets); l != 1 {
			t.Errorf("SetBucket() got = %v, want %v", l, 1)
		}
	})
	t.Run("print", func(t *testing.T) {
		if s := i.PrintBuckets(); s != bucket1 {
			t.Errorf("SetBucket() got = %v, want %v", s, bucket1)
		}
	})
}

func Test_SetSource(t *testing.T) {
	c := parse.Parser{}
	if err := c.SetSource(bucket1); err != nil {
		t.Errorf("SetSource(%v) returned the error: %v", bucket1, err)
	}
	t.Run("test set", func(t *testing.T) {
		if s := c.Source; s == "" {
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
