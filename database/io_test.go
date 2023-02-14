// © Ben Garrett https://github.com/bengarrett/dupers
package database_test

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/bengarrett/dupers/database"
	"github.com/bengarrett/dupers/internal/mock"
	"github.com/gookit/color"
	bolt "go.etcd.io/bbolt"
)

func init() { //nolint:gochecknoinits
	database.TestMode = true
}

func TestBackup(t *testing.T) {
	color.Enable = false
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"backup", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotWritten, err := database.Backup()
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
	color.Enable = false
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
			got, err := database.CopyFile(tt.args.src, tt.args.dest)
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

func TestCSVExport(t *testing.T) {
	color.Enable = false
	bucket1, err := mock.Bucket(1)
	if err != nil {
		t.Error(err)
	}
	DB, err := mock.Database()
	if err != nil {
		log.Panic(err)
	}
	defer DB.Close()
	t.Run("csv export", func(t *testing.T) {
		gotName, err := database.CSVExport(DB, bucket1)
		if err != nil {
			t.Errorf("Backup() error = %v, want nil", err)
			return
		}
		if gotName == "" {
			t.Errorf("Backup() gotName = \"\", want %s", bucket1)
		}
		if gotName != "" {
			if err := os.Remove(gotName); err != nil {
				log.Println(err)
			}
		}
	})
}

func TestImport(t *testing.T) {
	color.Enable = false
	exp1, err := mock.Export(1)
	if err != nil {
		t.Error(err)
	}
	DB, err := mock.Database()
	if err != nil {
		t.Error(err)
	}
	defer DB.Close()
	r, err := database.Import(DB, "", nil)
	if r != 0 {
		t.Errorf("Import(empty) records != 0")
	}
	if err == nil {
		t.Errorf("Import(empty) expect error, not nil")
	}
	openCSV, err := os.Open(exp1)
	if err != nil {
		t.Error(err)
	}
	defer openCSV.Close()
	bucket, ls, err := database.Scanner(openCSV)
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

func TestScanner(t *testing.T) {
	color.Enable = false
	item1, err := mock.Item(1)
	if err != nil {
		t.Error(err)
	}
	exp1, err := mock.Export(1)
	if err != nil {
		t.Error(err)
	}
	openBin, err := os.Open(item1)
	if err != nil {
		t.Error(err)
	}
	defer openBin.Close()
	openCSV, err := os.Open(exp1)
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
			got, got1, err := database.Scanner(tt.file)
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

func TestCSVImport(t *testing.T) {
	color.Enable = false
	DB, err := mock.Database()
	if err != nil {
		log.Panic(err)
	}
	defer DB.Close()
	type args struct {
		name string
		db   *bolt.DB
	}
	tests := []struct {
		name        string
		args        args
		wantRecords int
		wantErr     bool
	}{
		{"invalid", args{}, 0, true},
		{"no path", args{"", DB}, 0, true},
		{"only file", args{os.TempDir(), DB}, 0, true},
		{"okay", args{"../test/export-bucket1.csv", DB}, 26, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRecords, err := database.CSVImport(tt.args.db, tt.args.name, false)
			if (err != nil) != tt.wantErr {
				t.Errorf("CSVImport() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotRecords != tt.wantRecords {
				t.Errorf("CSVImport() = %v, want %v", gotRecords, tt.wantRecords)
			}
		})
	}
}
