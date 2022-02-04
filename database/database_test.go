package database_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/bengarrett/dupers/database"
	"github.com/bengarrett/dupers/internal/mock"
	"github.com/gookit/color"
)

const (
	testSrc = "../test/files_to_check/ppFlTD6QQYlS"
	testDst = "../test/tmp/ppFlTD6QQYlS"
)

func init() { //nolint:gochecknoinits
	color.Enable = false
	database.TestMode = true
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
			gotNames, err := database.AllBuckets(nil)
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

func TestClean(t *testing.T) {
	type args struct {
		quiet bool
		debug bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"1", args{quiet: false, debug: false}, false},
		{"2", args{quiet: true, debug: false}, false},
		{"3", args{quiet: false, debug: true}, true},
		{"4", args{quiet: true, debug: true}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := mock.TestOpen(); err != nil {
				t.Error(err)
				return
			}
			err := database.Clean(tt.args.quiet, tt.args.debug)
			if tt.args.quiet == true {
				if (err != nil) != tt.wantErr {
					t.Errorf("Clean() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				return
			}
			if tt.args.debug == true && !errors.Is(database.ErrDBClean, err) {
				t.Errorf("Clean() expected %v error, got %v", database.ErrDBClean, err)
				return
			}
			if (err != nil) != tt.wantErr {
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
			if err := database.Compact(false); (err != nil) != tt.wantErr {
				if !errors.As(err, &database.ErrDBCompact) {
					t.Errorf("Compact() error = %v, wantErr %t", err, tt.wantErr)
					return
				}
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
	empty, find := database.Matches{}, database.Matches{}
	item := mock.Item1()
	k := database.Filepath(item)
	find[k] = database.Bucket(mock.Bucket1())
	tests := []struct {
		name    string
		args    args
		want    *database.Matches
		wantErr bool
	}{
		{"match all", args{"UEvz", nil}, &find, false},
		{"exact", args{item, nil}, &find, false},
		{"no match", args{"abcde", nil}, &empty, false},
		{"upper", args{strings.ToUpper(item), nil}, &empty, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := database.Compare(tt.args.s, tt.args.buckets...)
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
	empty, find := database.Matches{}, database.Matches{}
	item := mock.Item1()
	s, k := filepath.Base(item), database.Filepath(item)
	find[k] = database.Bucket(mock.Bucket1())
	tests := []struct {
		name    string
		args    args
		want    *database.Matches
		wantErr bool
	}{
		{"match", args{s, nil}, &find, false},
		{"upper", args{strings.ToUpper(s), nil}, &empty, false},
		{"no match", args{"abcde", nil}, &empty, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := database.CompareBase(tt.args.s, tt.args.buckets...)
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
		want    *database.Matches
		wantErr bool
	}{
		{"match", args{s, nil}, &find, false},
		{"upper", args{strings.ToUpper(s), nil}, &find, false},
		{"no match", args{"abcde", nil}, &empty, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := database.CompareBaseNoCase(tt.args.s, tt.args.buckets...)
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
	empty, find := database.Matches{}, database.Matches{}
	item := mock.Item1()
	s, k := filepath.Base(item), database.Filepath(item)
	find[k] = database.Bucket(mock.Bucket1())
	tests := []struct {
		name    string
		args    args
		want    *database.Matches
		wantErr bool
	}{
		{"match", args{"vz", nil}, &find, false},
		{"exact", args{s, nil}, &find, false},
		{"no match", args{"abcde", nil}, &empty, false},
		{"upper", args{strings.ToUpper(s), nil}, &find, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := database.CompareNoCase(tt.args.s, tt.args.buckets...)
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

func TestExist(t *testing.T) {
	if err := mock.TestOpen(); err != nil {
		t.Error(err)
	}
	t.Run("exist", func(t *testing.T) {
		if err := database.Exist(mock.Bucket1(), nil); err != nil {
			t.Errorf("Exist() bucket1 error = %v, want nil", err)
		}
		if err := database.Exist(mock.Bucket2(), nil); err == nil {
			t.Error("Exist() bucket2 error = nil, want error")
		}
		if err := database.Exist("", nil); err == nil {
			t.Error("Exist() empty bucket error = nil, want error")
		}
	})
	if err := mock.TestRemove(); err != nil {
		log.Fatal(err)
	}
}

func TestIsEmpty(t *testing.T) {
	if err := mock.TestOpen(); err != nil {
		t.Error(err)
	}
	t.Run("is empty", func(t *testing.T) {
		// test db with bucket
		wantErr, want := false, false
		got, err := database.IsEmpty()
		if (err != nil) != wantErr {
			t.Errorf("IsEmpty() error = %v, wantErr %v", err, wantErr)
			return
		}
		if got != want {
			t.Errorf("IsEmpty() = %v, want %v", got, want)
		}
		// test & use remove bucket, leaving the db empty
		if err1 := database.RM(mock.Bucket1()); err1 != nil {
			fmt.Println(err1)
		}
		// test empty db
		wantErr, want = false, true
		got, err = database.IsEmpty()
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
			gotLs, err := database.List(tt.bucket, nil)
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

func TestInfo(t *testing.T) {
	if err := mock.TestOpen(); err != nil {
		t.Error(err)
	}
	info, err := database.Info()
	if err != nil {
		t.Errorf("Info() returned an error = %v", err)
	}
	if want := mock.Bucket1(); !strings.Contains(info, want) {
		t.Errorf("Info() should display the mock database path, %v\ngot:\n%v", want, info)
	}
	if err := mock.TestRemove(); err != nil {
		log.Fatal(err)
	}
}

func TestRename(t *testing.T) {
	if err := mock.TestOpen(); err != nil {
		t.Error(err)
	}
	t.Run("rename", func(t *testing.T) {
		if err := database.Rename(mock.Bucket2(), mock.Bucket1()); err == nil {
			t.Error("Rename() bucket2 to bucket1 error = nil, want error")
		}
		if err := database.Rename("", ""); err == nil {
			t.Error("Rename() empty buckets error = nil, want error")
		}
		if err := database.Rename(mock.Bucket1(), ""); err == nil {
			t.Error("Rename() empty new bucket error = nil, want error")
		}
		if err := database.Rename("", mock.Bucket2()); err == nil {
			t.Error("Rename() empty bucket error = nil, want error")
		}
	})
	if err := mock.TestRemove(); err != nil {
		log.Fatal(err)
	}
}

func TestCreate(t *testing.T) {
	tmp, err := ioutil.TempFile(os.TempDir(), "dupers_create_test.db")
	if err != nil {
		t.Error(err)
	}
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"empty", "", true},
		{"dir", ".", true},
		{"temp file", tmp.Name(), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := database.Create(tt.path); (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.name == "temp file" {
				defer os.Remove(tmp.Name())
			}
		})
	}
}

func TestDB(t *testing.T) {
	var (
		path string
		err  error
	)
	t.Run("sequence 1", func(t *testing.T) {
		path, err = database.DB()
		if err != nil {
			t.Errorf("DB() #1 error = %v", err)
			return
		}
		if (path == "") != false {
			t.Errorf("DB() returned an empty path")
		}
		if err := os.RemoveAll(path); err != nil {
			t.Errorf("DB RemoveAll() error = %v", err)
			return
		}
	})
	t.Run("sequence 2", func(t *testing.T) {
		path, err = database.DB()
		if err != nil {
			t.Errorf("DB() #2 error = %v", err)
			return
		}
		if (path == "") != false {
			t.Errorf("DB() returned an empty path")
			return
		}
	})
	t.Run("sequence 3", func(t *testing.T) {
		err = os.WriteFile(path, []byte(""), 0o600)
		if err != nil {
			t.Errorf("DB WriteFile() error = %v", err)
			return
		}
		s, err := os.Stat(path)
		if err != nil {
			t.Errorf("DB Stat error = %v", err)
			return
		}
		if s.Size() != 0 {
			t.Errorf("DB Stat error, expected a zero-byte file: %s", path)
		}
		path, err = database.DB()
		if err != nil {
			t.Errorf("DB() #3 error = %v", err)
			return
		}
		s, err = os.Stat(path)
		if err != nil {
			t.Errorf("DB Stat error = %v", err)
			return
		}
		fmt.Println("size", s.Size())
		if s.Size() == 0 {
			t.Errorf("DB Stat error, expected a new database file: %s", path)
			return
		}
	})
}
