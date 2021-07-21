// © Ben Garrett https://github.com/bengarrett/dupers

package dupers

import (
	"fmt"
	"path/filepath"
	"testing"
)

const (
	bucket = "../test/bucket1"
	// hash values created from sha256sum <filename>.
	file1 = "../test/bucket1/0vlLaUEvzAWP"
	hash1 = "1a1d76a3187ccee147e6c807277273afbad5d2680f5eadf1012310743e148f22"
	file2 = "../test/bucket1/GwejJkMzs3yP"
	hash2 = "4acc274c2e6dc2241029c735758f672b3dc1109ab76a91fe29aeb2bac6949eb7"
	hash0 = "0000000000000000000000000000000000000000000000000000000000000000"
)

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
			if got := contains(tt.args.s, tt.args.find); got != tt.want {
				t.Errorf("contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_containsBin(t *testing.T) {
	d, err := filepath.Abs(bucket)
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

// func Test_zread(t *testing.T) {
// 	tests := []struct {
// 		name     string
// 		path     string
// 		wantHash string
// 		wantErr  bool
// 	}{
// 		{"empty", "", hash0, true},
// 		{"file1", file1, hash1, false},
// 		{"file2", file2, hash2, false},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			gotHash, err := read(tt.path)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("read() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			h := fmt.Sprintf("%x", gotHash)
// 			if !reflect.DeepEqual(h, tt.wantHash) {
// 				t.Errorf("read() = %v, want %v", h, tt.wantHash)
// 			}
// 		})
// 	}
// }

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
