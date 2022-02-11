// © Ben Garrett https://github.com/bengarrett/dupers
package csv_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/bengarrett/dupers/database/internal/csv"
	"github.com/bengarrett/dupers/internal/mock"
)

func mockDir() string {
	sep := string(os.PathSeparator)
	dir := filepath.Join("home", "me", "Downloads")
	if runtime.GOOS == csv.WinOS {
		return filepath.Join("C:", sep, dir)
	}
	return filepath.Join(sep, dir)
}

func TestBucket(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{"empty", "", ""},
		{"invalid", "invalid header", ""},
		{"no path", csv.Header, ""},
		{"local path", csv.Header + mockDir(), mockDir()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := csv.Bucket(tt.s); got != tt.want {
				t.Errorf("Bucket() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChecker(t *testing.T) {
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
			if err := csv.Checker(tt.file); (err != nil) != tt.wantErr {
				t.Errorf("Checker() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChecksum(t *testing.T) {
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
			got, err := csv.Checksum(tt.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("Checksum() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if fmt.Sprintf("%x", got) != tt.want {
				t.Errorf("Checksum() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestImport(t *testing.T) {
	const (
		helloWorld = "68656c6c6f20776f726c64"
		sum        = "44dcc97a2b115c9fd51c95d6a3f2075f2f7c09067e34a33d9259cd22208bffba"
		file       = "someimage.png"
	)
	path := "/home/me/downloads"
	if runtime.GOOS == csv.WinOS {
		path = "C:\\home\\me\\downloads"
	}
	abs := strings.Join([]string{path, file}, string(os.PathSeparator))
	line := strings.Join([]string{sum, file}, ",")
	bsum, err := csv.Checksum(sum)
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
			gotSum, gotPath, err := csv.Import(tt.args.line, tt.args.bucket)
			if (err != nil) != tt.wantErr {
				t.Errorf("Import() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotSum, tt.wantSum) {
				t.Errorf("Import() gotSum = %v, want %v", gotSum, tt.wantSum)
			}
			if gotPath != tt.wantPath {
				t.Errorf("Import() gotPath = %v, want %v", gotPath, tt.wantPath)
			}
		})
	}
}

func TestPathWindows(t *testing.T) {
	unc := fmt.Sprintf("%sserver%sshare%s", csv.UncPath, csv.BackSlash, csv.BackSlash)
	tests := []struct {
		name string
		path string
		want string
	}{
		{"empty", "", ""},
		{"win drive", "C:", "C:"},
		{"win drive tail", "C:\\", "C:"},
		{"windows", "C:\\Users\\Ben\\Downloads\\", "C:\\Users\\Ben\\Downloads\\"},
		{"linux", "/home/ben/Downloads", "C:\\home\\ben\\Downloads"},
		{"linux tail", "/home/ben/Downloads/", "C:\\home\\ben\\Downloads\\"},
		{"root", "/", "C:"},
		{"unc", unc, unc},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := csv.PathWindows(tt.path); got != tt.want {
				t.Errorf("PathWindows() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPathPosix(t *testing.T) {
	unc := fmt.Sprintf("%sserver%sshare%s", csv.UncPath, csv.BackSlash, csv.BackSlash)
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
		{"network", unc, "/server/share/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := csv.PathPosix(tt.path); got != tt.want {
				t.Errorf("pathPosix() = %q, want %q", got, tt.want)
			}
		})
	}
}
