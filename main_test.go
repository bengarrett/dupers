package main

import (
	"fmt"
	"testing"

	"github.com/bengarrett/dupers/dupers"
	"github.com/gookit/color"
)

const (
	match   = "test/files_to_check"
	bucket1 = "test/bucket1"
	bucket2 = "test/bucket2"
)

func init() {
	color.Enable = false
}

func BenchmarkRM(*testing.B) {
	args := [2]string{"rm", bucket2}
	c := dupers.Config{Quiet: true, Test: true}
	taskDBUp(&c, false, args)
	taskDBRM(false, args)
}

func BenchmarkScan1(*testing.B) {
	args := []string{"dupe", match, bucket1}
	c := dupers.Config{Quiet: true, Test: true}
	var arr [2]string
	copy(arr[:], args)
	taskDBUp(&c, false, arr)
	f := false
	ts := tasks{
		lookup: &f,
		quiet:  &f,
		rm:     &f,
		sensen: &f,
	}
	taskScan(&c, ts, args...)
}

func BenchmarkScan2(*testing.B) {
	args := []string{"dupe", match, bucket1}
	c := dupers.Config{Quiet: true, Test: true}
	f, t := false, true
	ts := tasks{
		lookup: &f,
		quiet:  &t,
		rm:     &f,
		sensen: &f,
	}
	var arr [2]string
	copy(arr[:], args)
	taskDBUp(&c, false, arr)
	taskScan(&c, ts, args...)
}

func BenchmarkSearch(*testing.B) {
	terms := []string{"TzgPJuhfPJlg", "hello worlld"}
	const bucket = bucket1
	for _, term := range terms {
		args := []string{"search", fmt.Sprintf("'%s'", term), bucket}
		c := dupers.Config{Quiet: true, Test: true}
		f := false
		ts := tasks{
			exact:    &f,
			filename: &f,
			quiet:    &f,
		}
		var arr [2]string
		copy(arr[:], args)
		taskDBUp(&c, false, arr)
		for i := 0; i <= 3; i++ {
			taskSearch(ts, args...)
		}
	}
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
			if got := searchSummary(tt.args.total, tt.args.term, tt.args.exact, tt.args.filename); got != tt.want {
				t.Errorf("searchSummary() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_info(t *testing.T) {
	t.Run("info", func(t *testing.T) {
		if info() == "" {
			t.Error("info() = \"\", want strings")
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
