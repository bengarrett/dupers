package main

import (
	"fmt"
	"testing"

	dupers "github.com/bengarrett/dupers/lib"

	"github.com/gookit/color"
)

const (
	match   = "test/files_to_check"
	bucket  = "test/bucket1"
	bucket2 = "test/bucket2"
)

func BenchmarkRM(*testing.B) {
	color.Enable = false
	args := []string{"rm", bucket2}
	c := dupers.Config{Quiet: true, Test: true}
	taskDBUp(&c, args...)
	taskDBRM(false, args...)
}

func BenchmarkScan1(*testing.B) {
	color.Enable = false
	args := []string{"dupe", match, bucket}
	c := dupers.Config{Quiet: true, Test: true}
	taskDBUp(&c, args[1:]...)
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
	color.Enable = false
	args := []string{"dupe", match, bucket}
	c := dupers.Config{Quiet: true, Test: true}
	f, t := false, true
	ts := tasks{
		lookup: &f,
		quiet:  &t,
		rm:     &f,
		sensen: &f,
	}
	taskDBUp(&c, args[1:]...)
	taskScan(&c, ts, args...)
}

func BenchmarkSearch1(*testing.B) {
	color.Enable = false
	const term, bucket = "hello world", bucket
	args := []string{"search", fmt.Sprintf("'%s'", term), bucket}
	c := dupers.Config{Quiet: true, Test: true}
	f := false
	ts := tasks{
		exact:    &f,
		filename: &f,
		quiet:    &f,
	}
	taskDBUp(&c, args[1:]...)
	for i := 0; i <= 3; i++ {
		taskSearch(ts, args...)
	}
}

func BenchmarkSearch2(*testing.B) {
	color.Enable = false
	const term, bucket = "TzgPJuhfPJlg", bucket
	args := []string{"search", fmt.Sprintf("'%s'", term), bucket}
	c := dupers.Config{Quiet: true, Test: true}
	f := false
	ts := tasks{
		exact:    &f,
		filename: &f,
		quiet:    &f,
	}
	taskDBUp(&c, args[1:]...)
	for i := 0; i <= 3; i++ {
		taskSearch(ts, args...)
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
	color.Enable = false
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := searchSummary(tt.args.total, tt.args.term, tt.args.exact, tt.args.filename); got != tt.want {
				t.Errorf("searchSummary() = %v, want %v", got, tt.want)
			}
		})
	}
}
