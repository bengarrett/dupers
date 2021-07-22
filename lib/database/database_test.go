package database

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gookit/color"
	bolt "go.etcd.io/bbolt"
)

const (
	fileSrc  = "../../test/files_to_check/ppFlTD6QQYlS"
	fileDest = "../../test/tmp/ppFlTD6QQYlS"

	bucket = "test/bucket1"
	key1   = "item1"
	val1   = "some value 1"
)

var (
	ErrBucket = errors.New("bucket already exists")
	ErrCreate = errors.New("create bucket")
	ErrNoComp = errors.New("database compression has not reduced the size")
)

func tmpBk() string {
	b, err := filepath.Abs(bucket)
	if err != nil {
		log.Fatal(err)
	}
	return b
}

func tmpDB() error {
	testMode = true
	path, err := DB()
	if err != nil {
		return err
	}
	db, err := bolt.Open(path, FileMode, nil)
	if err != nil {
		return err
	}
	defer db.Close()
	return db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucket([]byte(tmpBk()))
		if err != nil {
			if errors.As(err, &ErrBucket) {
				return nil
			}
			return fmt.Errorf("%w: %s", ErrCreate, err)
		}
		return b.Put([]byte(key1), []byte(val1))
	})
}

func tmpRM() error {
	testMode = true
	path, err := DB()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return nil
}

func Test_copyFile(t *testing.T) {
	type args struct {
		src  string
		dest string
	}
	d, err := filepath.Abs(fileDest)
	if err != nil {
		t.Error(err)
	}
	s, err := filepath.Abs(fileSrc)
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
			got, err := copyFile(tt.args.src, tt.args.dest)
			if (err != nil) != tt.wantErr {
				t.Errorf("copyFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("copyFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuckets(t *testing.T) {
	color.Enable = false
	testMode = true
	tests := []struct {
		name      string
		wantNames []string
		wantErr   bool
	}{
		{"test", []string{tmpBk()}, false},
	}
	if err := tmpDB(); err != nil {
		t.Error(err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNames, err := Buckets()
			if (err != nil) != tt.wantErr {
				t.Errorf("Buckets() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotNames, tt.wantNames) {
				t.Errorf("Buckets() = %v, want %v", gotNames, tt.wantNames)
			}
		})
	}
}

func TestBackup(t *testing.T) {
	color.Enable = false
	testMode = true
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"backup", false},
	}
	if err := tmpDB(); err != nil {
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
	if err := tmpDB(); err != nil {
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
	if err := tmpDB(); err != nil {
		t.Error(err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Compact(false); (err != nil) != tt.wantErr {
				if errors.As(err, &ErrNoComp) {
					return
				}
				t.Errorf("Compact() error = %v, wantErr %t", err, tt.wantErr)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	if err := tmpRM(); err != nil {
		log.Fatal(err)
	}
	if err := tmpDB(); err != nil {
		t.Error(err)
	}
	i, err := Info()
	if err != nil {
		t.Error(err)
	}
	fmt.Println(i)
	type args struct {
		s       string
		buckets []string
	}
	empty, find := Matches{}, Matches{}
	find[key1] = Bucket(tmpBk())
	tests := []struct {
		name    string
		args    args
		want    *Matches
		wantErr bool
	}{
		{"match", args{"item", nil}, &find, false},
		{"exact", args{key1, nil}, &find, false},
		{"no match", args{"abcde", nil}, &empty, false},
		{"upper", args{strings.ToUpper(key1), nil}, &empty, false},
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
	if err := tmpDB(); err != nil {
		t.Error(err)
	}
	i, err := Info()
	if err != nil {
		t.Error(err)
	}
	fmt.Println(i)
	type args struct {
		s       string
		buckets []string
	}
	empty, find := Matches{}, Matches{}
	find[key1] = Bucket(tmpBk())
	tests := []struct {
		name    string
		args    args
		want    *Matches
		wantErr bool
	}{
		{"match", args{key1, nil}, &find, false},
		{"upper", args{strings.ToUpper(key1), nil}, &empty, false},
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
		{"match", args{key1, nil}, &find, false},
		{"upper", args{strings.ToUpper(key1), nil}, &find, false},
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
	if err := tmpDB(); err != nil {
		t.Error(err)
	}
	type args struct {
		s       string
		buckets []string
	}
	empty, find := Matches{}, Matches{}
	find[key1] = Bucket(tmpBk())
	tests := []struct {
		name    string
		args    args
		want    *Matches
		wantErr bool
	}{
		{"match", args{"item", nil}, &find, false},
		{"exact", args{key1, nil}, &find, false},
		{"no match", args{"abcde", nil}, &empty, false},
		{"upper", args{strings.ToUpper(key1), nil}, &find, false},
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
	if err := tmpDB(); err != nil {
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
		if err1 := RM(tmpBk()); err1 != nil {
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
		// delete modified db
		if err := tmpRM(); err != nil {
			log.Fatal(err)
		}
	})
}
