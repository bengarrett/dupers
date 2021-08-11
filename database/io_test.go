// © Ben Garrett https://github.com/bengarrett/dupers

// Package database interacts with Dupers bbolt database and buckets.
package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/bengarrett/dupers/mock"
	"github.com/gookit/color"
)

func init() {
	color.Enable = false
	testMode = true
}

func TestBackup(t *testing.T) {
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

func TestExportCSV(t *testing.T) {
	color.Enable = false
	if err := mock.TestOpen(); err != nil {
		t.Error(err)
	}
	t.Run("csv export", func(t *testing.T) {
		gotName, err := ExportCSV(mock.Bucket1(), nil)
		if err != nil {
			t.Errorf("Backup() error = %v, want nil", err)
			return
		}
		if gotName == "" {
			t.Errorf("Backup() gotName = \"\"")
		}
		if gotName != "" {
			if err := os.Remove(gotName); err != nil {
				log.Println(err)
			}
		}
	})
}

func TestImportCSV(t *testing.T) {
	db, err := mock.Open()
	if err != nil {
		t.Error(err)
	}
	defer db.Close()
	r, err := ImportCSV("", db)
	if r != 0 {
		t.Errorf("ImportCSV(empty) records != 0")
	}
	if err == nil {
		t.Errorf("ImportCSV(empty) expect error, not nil")
	}
	r, err = ImportCSV(mock.Export1(), db)
	if r != 26 {
		t.Errorf("ImportCSV(export1) expect 26 records to be imported")
	}
	if err != nil {
		t.Errorf("ImportCSV(export1) expect no error, not %v", err)
	}
}

func Test_csvChecker(t *testing.T) {
	openBin, err := os.Open(mock.Item1())
	if err != nil {
		t.Error(err)
	}
	defer openBin.Close()
	openCSV, err := os.Open(mock.Export1())
	if err != nil {
		t.Error(err)
	}
	defer openCSV.Close()
	tests := []struct {
		name    string
		file    *os.File
		wantErr bool
	}{
		{"empty", nil, true},
		{"binary file", openBin, true},
		{"csv file", openCSV, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := csvChecker(tt.file); (err != nil) != tt.wantErr {
				t.Errorf("csvChecker() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_csvScanner(t *testing.T) {
	openBin, err := os.Open(mock.Item1())
	if err != nil {
		t.Error(err)
	}
	defer openBin.Close()
	openCSV, err := os.Open(mock.Export1())
	if err != nil {
		t.Error(err)
	}
	defer openCSV.Close()
	tests := []struct {
		name       string
		file       *os.File
		wantBucket bool
		wantLists  int
		wantErr    bool
	}{
		{"empty", nil, false, 0, true},
		{"binary file", openBin, false, 0, true},
		{"csv file", openCSV, true, 26, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := csvScanner(tt.file)
			if (err != nil) != tt.wantErr {
				t.Errorf("csvScanner() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (len(got) > 0) != tt.wantBucket {
				t.Errorf("csvScanner() wantBucket got = %v, want %v", (len(got) > 0), tt.wantBucket)
			}
			if tt.wantLists == 0 && got1 != nil {
				t.Errorf("csvScanner() got1 = %v, want nil", got1)
			} else if tt.wantLists > 0 && len(*got1) != tt.wantLists {
				t.Errorf("csvScanner() got1 = %v, want %v", len(*got1), tt.wantLists)
			}
		})
	}
}

func mockDir() string {
	dir := filepath.Join("home", "me", "Downloads")
	if runtime.GOOS == "Windows" {
		filepath.Join("C:", dir)
	}
	return filepath.Join(string(os.PathSeparator), dir)
}

func Test_bucketName(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{"empty", "", ""},
		{"invalid", "invalid header", ""},
		{"no path", csvHeader, ""},
		{"local path", csvHeader + mockDir(), mockDir()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bucketName(tt.s); got != tt.want {
				t.Errorf("bucketName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_winPosix(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"empty", "", ""},
		{"linux", "/home/ben/Downloads", "/home/ben/Downloads"},
		{"linux tail", "/home/ben/Downloads/", "/home/ben/Downloads/"},
		{"drive", "C:", "/"},
		{"windows", "C:\\Users\\Ben\\Downloads\\", "/Users/Ben/Downloads/"},
		{"network", fmt.Sprintf("%sserver%sshare%s", uncPath, backslash, backslash), "/server/share/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := winPosix(tt.path); got != tt.want {
				t.Errorf("winPosix() = %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_importCSV(t *testing.T) {
	const (
		helloWorld = "68656c6c6f20776f726c64"
		sum        = "44dcc97a2b115c9fd51c95d6a3f2075f2f7c09067e34a33d9259cd22208bffba"
		path       = "/home/me/downloads"
		file       = "someimage.png"
	)
	abs := strings.Join([]string{path, file}, fwdslash)
	line := strings.Join([]string{sum, file}, ",")
	//invalid := strings.Join([]string{sum, ""}, ",")
	bsum, err := checksum(sum)
	if err != nil {
		t.Error(err)
	}
	empty := [32]byte{}
	type args struct {
		line   string
		bucket string
	}
	tests := []struct {
		name     string
		args     args
		wantSum  [32]byte
		wantPath string
		wantErr  bool
	}{
		{"empty", args{"", ""}, empty, "", true},
		{"invalid hash", args{helloWorld, ""}, empty, "", true},
		{"invalid data", args{"invalid,csv data", path}, empty, "", true},
		{"empty path", args{sum + ",", path}, bsum, path, false},
		{"valid hash", args{line, path}, bsum, abs, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSum, gotPath, err := importCSV(tt.args.line, tt.args.bucket)
			if (err != nil) != tt.wantErr {
				t.Errorf("importCSV() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotSum, tt.wantSum) {
				t.Errorf("importCSV() gotSum = %v, want %v", gotSum, tt.wantSum)
			}
			if gotPath != tt.wantPath {
				t.Errorf("importCSV() gotPath = %v, want %v", gotPath, tt.wantPath)
			}
		})
	}
}

func Test_checksum(t *testing.T) {
	const (
		sum  = "44dcc97a2b115c9fd51c95d6a3f2075f2f7c09067e34a33d9259cd22208bffba"
		null = "0000000000000000000000000000000000000000000000000000000000000000"
	)
	tests := []struct {
		name    string
		s       string
		want    string
		wantErr bool
	}{
		{"empty", "", null, true},
		{"null", null, null, false},
		{"sum", sum, sum, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checksum(tt.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("checksum() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if fmt.Sprintf("%x", got) != tt.want {
				t.Errorf("checksum() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestImport(t *testing.T) {
	db, err := mock.Open()
	if err != nil {
		t.Error(err)
	}
	defer db.Close()
	r, err := Import("", nil, db)
	if r != 0 {
		t.Errorf("Import(empty) records != 0")
	}
	if err == nil {
		t.Errorf("Import(empty) expect error, not nil")
	}
	openCSV, err := os.Open(mock.Export1())
	if err != nil {
		t.Error(err)
	}
	defer openCSV.Close()
	bucket, ls, err := csvScanner(openCSV)
	if err != nil {
		t.Errorf("Import(csv) unexpected error, %v", err)
	}
	if bucket == "" {
		t.Error("Import(csv) unexpected empty bucket name")
	}
	if ls == nil || len(*ls) != 26 {
		t.Errorf("Import(csv) List is empty")
	}
}
