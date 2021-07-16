// © Ben Garrett https://github.com/bengarrett/dupers

// dupers todo
package dupers

import (
	"path/filepath"
	"testing"
)

const (
	bucket = "../test/bucket1"
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
