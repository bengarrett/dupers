// © Ben Garrett https://github.com/bengarrett/dupers
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
	bolt "go.etcd.io/bbolt"
)

const (
	testSrc = "../test/files_to_check/ppFlTD6QQYlS"
	testDst = "../test/tmp/ppFlTD6QQYlS"
	test0b  = "../test/zerobytefile"
)

func init() { //nolint:gochecknoinits
	color.Enable = false
	database.TestMode = true
	if err := mock.TestRemove(); err != nil {
		log.Fatal(err)
	}
}

func TestAbs(t *testing.T) {
	tests := []struct {
		name      string
		wantEmpty bool
		wantErr   bool
	}{
		{"", false, false},
		{"abc", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := database.Abs(tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("Abs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (got == "") != tt.wantEmpty {
				t.Errorf("Abs() = %v, want %v", got, tt.wantEmpty)
			}
		})
	}
}

func TestAll(t *testing.T) {
	color.Enable = false
	b1, err := mock.Bucket1()
	if err != nil {
		t.Error(err)
	}
	tests := []struct {
		name     string
		wantName string
		wantErr  bool
	}{
		{"test", b1, false},
	}
	if err := mock.TestOpen(); err != nil {
		t.Error(err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNames, err := database.All(nil)
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

func TestCheck(t *testing.T) {
	if err := database.Check(""); err != nil {
		t.Errorf("Check() error = %v", err)
	}
	const invalid = "invalidpathdoesnotexist"
	if err := database.Check(invalid); err != database.ErrDBNotFound {
		t.Errorf("Check() error = %v, want %v", err, database.ErrDBNotFound)
	}
	b0, err := filepath.Abs(test0b)
	if err != nil {
		t.Error(err)
		return
	}
	if err := database.Check(b0); err != database.ErrDBZeroByte {
		t.Errorf("Check() error = %v, want %v", err, database.ErrDBZeroByte)
	}
}

func TestExist(t *testing.T) {
	t.Run("exist", func(t *testing.T) {
		if err := mock.TestOpen(); err != nil {
			t.Error(err)
		}
		b1, err := mock.Bucket1()
		if err != nil {
			t.Errorf("Exist() bucket1 error = %v, want nil", err)
		}
		if err := database.Exist(b1, nil); err != nil {
			t.Errorf("Exist() bucket1 error = %v, want nil", err)
		}
		b2, err := mock.Bucket2()
		if err != nil {
			t.Errorf("Exist() bucket1 error = %v, want nil", err)
		}
		if err := database.Exist(b2, nil); err == nil {
			t.Error("Exist() bucket2 error = nil, want error")
		}
		if err := database.Exist("", nil); err == nil {
			t.Error("Exist() empty bucket error = nil, want error")
		}
	})
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
			const debugOutput = true
			if err := database.Compact(debugOutput); (err != nil) != tt.wantErr {
				if !errors.Is(err, database.ErrDBCompact) {
					t.Errorf("Compact() error = %v, wantErr %t", err, tt.wantErr)
					return
				}
			}
		})
	}
}

func TestCompare(t *testing.T) {
	type args struct {
		s       string
		buckets []string
	}
	empty, find := database.Matches{}, database.Matches{}
	item, err := mock.Item1()
	if err != nil {
		t.Error(err)
	}
	b1, err := mock.Bucket1()
	if err != nil {
		t.Error(err)
	}
	k := database.Filepath(item)
	find[k] = database.Bucket(b1)
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
		if err := mock.TestOpen(); err != nil {
			t.Error(err)
			return
		}
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

func TestCompareBase(t *testing.T) { // nolint:funlen
	if err := mock.TestRemove(); err != nil {
		t.Error(err)
		return
	}
	type args struct {
		s       string
		buckets []string
	}
	empty, find := database.Matches{}, database.Matches{}
	item, err := mock.Item1()
	if err != nil {
		t.Error(err)
	}
	b1, err := mock.Bucket1()
	if err != nil {
		t.Error(err)
	}
	s, k := filepath.Base(item), database.Filepath(item)
	find[k] = database.Bucket(b1)
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
		if err := mock.TestOpen(); err != nil {
			t.Error(err)
			return
		}
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
	if err := mock.TestOpen(); err != nil {
		t.Error(err)
		return
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
	b1, err := mock.Bucket1()
	if err != nil {
		t.Error(err)
	}
	item, err := mock.Item1()
	if err != nil {
		t.Error(err)
	}
	s, k := filepath.Base(item), database.Filepath(item)
	find[k] = database.Bucket(b1)
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

func TestCount(t *testing.T) {
	db, err := mock.Open()
	if err != nil {
		t.Error(err)
	}
	defer db.Close()
	b1, err := mock.Bucket1()
	if err != nil {
		t.Error(err)
	}
	b2, err := mock.Bucket2()
	if err != nil {
		t.Error(err)
	}
	if err := mock.CreateItem(b2, test0b, db); err != nil {
		t.Error(err)
	}
	type args struct {
		name string
		db   *bolt.DB
	}
	tests := []struct {
		name      string
		args      args
		wantItems int
		wantErr   bool
	}{
		{"nil db", args{"", nil}, 0, true},
		{"empty name", args{"", db}, 0, true},
		{"bucket #1", args{b1, db}, 1, false},
		{"bucket #2", args{b2, db}, 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotItems, err := database.Count(tt.args.name, tt.args.db)
			if (err != nil) != tt.wantErr {
				t.Errorf("Count() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotItems != tt.wantItems {
				t.Errorf("Count() = %v, want %v", gotItems, tt.wantItems)
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

func TestInfo(t *testing.T) {
	if err := mock.TestOpen(); err != nil {
		t.Error(err)
	}
	info, err := database.Info("")
	if err != nil {
		t.Errorf("Info() returned an error = %v", err)
	}
	b1, err := mock.Bucket1()
	if err != nil {
		t.Error(err)
	}
	if want := b1; !strings.Contains(info, want) {
		t.Errorf("Info() should display the mock database path, %v\ngot:\n%v", want, info)
	}
	_, err = database.Info(test0b + "placeholderfiller")
	if err != database.ErrDBNotFound {
		t.Errorf("Info() not found test should return, %v, got %v", database.ErrDBNotFound, err)
	}
}

func TestIsEmpty(t *testing.T) {
	t.Run("is empty", func(t *testing.T) {
		if err := mock.TestOpen(); err != nil {
			t.Error(err)
			return
		}
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
		b1, err := mock.Bucket1()
		if err != nil {
			t.Error(err)
		}
		// test & use remove bucket, leaving the db empty
		if err1 := database.RM(b1); err1 != nil {
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
	})
}

func TestList(t *testing.T) {
	if err := mock.TestOpen(); err != nil {
		t.Error(err)
	}
	b1, err := mock.Bucket1()
	if err != nil {
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
		{"backet", b1, true, false},
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
}

func TestRename(t *testing.T) {
	if err := mock.TestOpen(); err != nil {
		t.Error(err)
	}
	b1, err := mock.Bucket1()
	if err != nil {
		t.Error(err)
	}
	b2, err := mock.Bucket2()
	if err != nil {
		t.Error(err)
	}
	t.Run("rename", func(t *testing.T) {
		if err := database.Rename(b2, b1); err == nil {
			t.Error("Rename() bucket2 to bucket1 error = nil, want error")
		}
		if err := database.Rename("", ""); err == nil {
			t.Error("Rename() empty buckets error = nil, want error")
		}
		if err := database.Rename(b1, ""); err == nil {
			t.Error("Rename() empty new bucket error = nil, want error")
		}
		if err := database.Rename("", b2); err == nil {
			t.Error("Rename() empty bucket error = nil, want error")
		}
	})
}
