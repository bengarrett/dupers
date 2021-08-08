package main

import (
	"testing"

	"github.com/gookit/color"
)

func init() {
	color.Enable = false
}

func Test_self(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"expected", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := self()
			if (err != nil) != tt.wantErr {
				t.Errorf("self() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_searchSummary(t *testing.T) {
	type args struct {
		total    int
		term     string
		exact    bool
		filename bool
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"empty", args{}, "No results exist for ''."},
		{"single", args{total: 1}, "1 result exist for ''."},
		{"two", args{total: 2}, "2 results exist for ''."},
		{"multiple results", args{term: "xyz", total: 3}, "3 results exist for 'xyz'."},
		{"exact results", args{term: "xyz", total: 3, exact: true}, "3 exact results exist for 'xyz'."},
		{"filename results", args{term: "xyz", total: 3, filename: true}, "3 filename results exist for 'xyz'."},
		{"exact filename results", args{term: "xyz", total: 3, exact: true, filename: true}, "3 exact filename results exist for 'xyz'."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := searchCmdSummary(tt.args.total, tt.args.term, tt.args.exact, tt.args.filename); got != tt.want {
				t.Errorf("searchCmdSummary() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_vers(t *testing.T) {
	t.Run("vers", func(t *testing.T) {
		if vers() == "" {
			t.Error("vers() = \"\", want strings")
		}
	})
}

func Test_home(t *testing.T) {
	t.Run("home", func(t *testing.T) {
		if home() == "" {
			t.Error("home() = \"\", want a directory path")
		}
	})
}
