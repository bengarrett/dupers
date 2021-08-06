package database

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/bengarrett/dupers/mock"
	"github.com/gookit/color"
)

const (
	testSrc = "../test/files_to_check/ppFlTD6QQYlS"
	testDst = "../test/tmp/ppFlTD6QQYlS"
)

func init() {
	color.Enable = false
	testMode = true
}

func TestCopyFile(t *testing.T) {
	type args struct {
		src  string
		dest string
	}
	d, err := filepath.Abs(testDst)
	if err != nil {
		t.Error(err)
	}
	s, err := filepath.Abs(testSrc)
	if err != nil {
		t.Error(err)
	}
	tests := []struct {
		name    string
		args    args
		want    int64
		wantErr bool
	}{
		{"empty", args{}, 0, true},
		{"exe", args{src: s, dest: d}, 5120, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CopyFile(tt.args.src, tt.args.dest)
			if (err != nil) != tt.wantErr {
				t.Errorf("CopyFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CopyFile() = %v, want %v", got, tt.want)
			}
		})
	}
	os.Remove(testDst)
}

func TestAllBuckets(t *testing.T) {
	color.Enable = false
	tests := []struct {
		name     string
		wantName string
		wantErr  bool
	}{
		{"test", mock.Bucket1(), false},
	}
	if err := mock.TestOpen(); err != nil {
		t.Error(err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNames, err := AllBuckets(nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("AllBuckets() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			sort.Strings(gotNames)
			if sort.SearchStrings(gotNames, tt.wantName) > len(gotNames) {
				t.Errorf("AllBuckets() = %v, want %v", gotNames, tt.wantName)
			}
		})
	}
}

func TestBackup(t *testing.T) {
	color.Enable = false
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"backup", false},
	}
	if err := mock.TestOpen(); err != nil {
		t.Error(err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotWritten, err := Backup()
			if (err != nil) != tt.wantErr {
				t.Errorf("Backup() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotName == "" {
				t.Errorf("Backup() gotName = \"\"")
			}
			if gotWritten == 0 {
				t.Errorf("Backup() gotWritten = %v, want something higher", gotWritten)
			}
			if gotName != "" {
				if err := os.Remove(gotName); err != nil {
					log.Println(err)
				}
			}
		})
	}
}

func TestClean(t *testing.T) {
	type args struct {
		quiet bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"temp", args{quiet: true}, false},
	}
	if err := mock.TestOpen(); err != nil {
		t.Error(err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Clean(tt.args.quiet, false); (err != nil) != tt.wantErr {
				t.Errorf("Clean() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCompact(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"temp", false},
	}
	if err := mock.TestOpen(); err != nil {
		t.Error(err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Compact(false); (err != nil) != tt.wantErr {
				if errors.As(err, &ErrDBCompact) {
					return
				}
				t.Errorf("Compact() error = %v, wantErr %t", err, tt.wantErr)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	if err := mock.TestRemove(); err != nil {
		log.Fatal(err)
	}
	if err := mock.TestOpen(); err != nil {
		t.Error(err)
	}
	type args struct {
		s       string
		buckets []string
	}
	empty, find := Matches{}, Matches{}
	item := mock.Item1()
	k := Filepath(item)
	find[k] = Bucket(mock.Bucket1())
	tests := []struct {
		name    string
		args    args
		want    *Matches
		wantErr bool
	}{
		{"match all", args{"UEvz", nil}, &find, false},
		{"exact", args{item, nil}, &find, false},
		{"no match", args{"abcde", nil}, &empty, false},
		{"upper", args{strings.ToUpper(item), nil}, &empty, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Compare(tt.args.s, tt.args.buckets...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Compare() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Compare() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareBase(t *testing.T) {
	if err := mock.TestOpen(); err != nil {
		t.Error(err)
	}
	type args struct {
		s       string
		buckets []string
	}
	empty, find := Matches{}, Matches{}
	item := mock.Item1()
	s, k := filepath.Base(item), Filepath(item)
	find[k] = Bucket(mock.Bucket1())
	tests := []struct {
		name    string
		args    args
		want    *Matches
		wantErr bool
	}{
		{"match", args{s, nil}, &find, false},
		{"upper", args{strings.ToUpper(s), nil}, &empty, false},
		{"no match", args{"abcde", nil}, &empty, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CompareBase(tt.args.s, tt.args.buckets...)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompareBase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CompareBase() = %v, want %v", got, tt.want)
			}
		})
	}
	tests = []struct {
		name    string
		args    args
		want    *Matches
		wantErr bool
	}{
		{"match", args{s, nil}, &find, false},
		{"upper", args{strings.ToUpper(s), nil}, &find, false},
		{"no match", args{"abcde", nil}, &empty, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CompareBaseNoCase(tt.args.s, tt.args.buckets...)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompareBaseNoCase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CompareBaseNoCase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareNoCase(t *testing.T) {
	if err := mock.TestOpen(); err != nil {
		t.Error(err)
	}
	type args struct {
		s       string
		buckets []string
	}
	empty, find := Matches{}, Matches{}
	item := mock.Item1()
	s, k := filepath.Base(item), Filepath(item)
	find[k] = Bucket(mock.Bucket1())
	tests := []struct {
		name    string
		args    args
		want    *Matches
		wantErr bool
	}{
		{"match", args{"vz", nil}, &find, false},
		{"exact", args{s, nil}, &find, false},
		{"no match", args{"abcde", nil}, &empty, false},
		{"upper", args{strings.ToUpper(s), nil}, &find, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CompareNoCase(tt.args.s, tt.args.buckets...)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompareNoCase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CompareNoCase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsEmpty(t *testing.T) {
	if err := mock.TestOpen(); err != nil {
		t.Error(err)
	}
	t.Run("is empty", func(t *testing.T) {
		// test db with bucket
		wantErr, want := false, false
		got, err := IsEmpty()
		if (err != nil) != wantErr {
			t.Errorf("IsEmpty() error = %v, wantErr %v", err, wantErr)
			return
		}
		if got != want {
			t.Errorf("IsEmpty() = %v, want %v", got, want)
		}
		// test & use remove bucket, leaving the db empty
		if err1 := RM(mock.Bucket1()); err1 != nil {
			t.Error(err1)
		}
		// test empty db
		wantErr, want = false, true
		got, err = IsEmpty()
		if (err != nil) != wantErr {
			t.Errorf("IsEmpty() error = %v, wantErr %v", err, wantErr)
			return
		}
		if got != want {
			t.Errorf("IsEmpty() = %v, want %v", got, want)
		}
		if err := mock.TestRemove(); err != nil {
			log.Fatal(err)
		}
	})
}

func TestList(t *testing.T) {
	if err := mock.TestOpen(); err != nil {
		t.Error(err)
	}
	tests := []struct {
		name    string
		bucket  string
		wantLs  bool
		wantErr bool
	}{
		{"empty", "", false, true},
		{"invalid", "foo bucket", false, true},
		{"backet", mock.Bucket1(), true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLs, err := List(tt.bucket, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if l := len(gotLs) > 0; l != tt.wantLs {
				t.Errorf("List() = %v, want %v", l, tt.wantLs)
			}
		})
	}
	// delete modified db
	if err := mock.TestRemove(); err != nil {
		log.Fatal(err)
	}
}

func TestSeek(t *testing.T) {
	sum0 := [32]byte{}
	sum1, err := mock.Read(mock.Item1())
	if err != nil {
		t.Error(err)
	}
	db, err := mock.Open()
	if err != nil {
		t.Error(err)
	}
	defer db.Close()
	if err := mock.CreateItem(mock.Bucket1(), mock.Item1(), db); err != nil {
		t.Error(err)
	}
	type args struct {
		sum    [32]byte
		bucket string
	}
	tests := []struct {
		name        string
		args        args
		wantFinds   []string
		wantRecords int
		wantErr     bool
	}{
		{"empty", args{sum0, ""}, nil, 0, true},
		{"no find", args{sum0, mock.Bucket1()}, nil, 1, false},
		{"find", args{sum1, mock.Bucket1()}, []string{mock.Item1()}, 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFinds, gotRecords, err := Seek(tt.args.sum, tt.args.bucket, db)
			if (err != nil) != tt.wantErr {
				t.Errorf("Seek() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotFinds, tt.wantFinds) {
				t.Errorf("Seek() gotFinds = %v, want %v", gotFinds, tt.wantFinds)
			}
			if gotRecords != tt.wantRecords {
				t.Errorf("Seek() gotRecords = %v, want %v", gotRecords, tt.wantRecords)
			}
		})
	}
}

func TestInfo(t *testing.T) {
	if err := mock.TestOpen(); err != nil {
		t.Error(err)
	}
	info, err := Info()
	if err != nil {
		t.Errorf("Info() returned an error = %v", err)
	}
	want := mock.Bucket1()
	if !strings.Contains(info, want) {
		t.Errorf("Info() should display the mock database path, %v\ngot:\n%v", want, info)
	}
	if err := mock.TestRemove(); err != nil {
		log.Fatal(err)
	}
}
