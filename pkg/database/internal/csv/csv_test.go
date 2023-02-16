// © Ben Garrett https://github.com/bengarrett/dupers
package csv_test

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/bengarrett/dupers/internal/mock"
	"github.com/bengarrett/dupers/pkg/database/internal/csv"
	"github.com/stretchr/testify/assert"
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
	s := csv.Bucket("")
	assert.Equal(t, "", s)

	const badHeader = "this is an invalid csv file header"
	s = csv.Bucket(badHeader)
	assert.Equal(t, "", s)

	s = csv.Bucket(csv.Header)
	assert.Equal(t, "", s, "header is missing directory info and should return an empty string")

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	s = csv.Bucket(csv.Header + bucket1)
	assert.Equal(t, bucket1, s)
}

func TestChecker(t *testing.T) {
	err := csv.Checker(nil)
	assert.NotNil(t, err)

	item1, err := mock.Item(1)
	assert.Nil(t, err)
	binary, err := os.Open(item1)
	assert.Nil(t, err)
	defer binary.Close()
	err = csv.Checker(binary)
	assert.NotNil(t, err, "binary files should return an error")

	text, err := os.Open(mock.CSV())
	assert.Nil(t, err)
	defer binary.Close()
	err = csv.Checker(text)
	assert.Nil(t, err)
}

func TestChecksum(t *testing.T) {
	var empty [32]byte
	zeros := strings.Repeat("0", 32)

	b, err := csv.Checksum("")
	assert.NotNil(t, err)
	assert.Equal(t, empty, b)

	b, err = csv.Checksum(zeros)
	assert.NotNil(t, err)
	assert.Equal(t, empty, b)

	sum, err := mock.ItemSum(1)
	assert.Nil(t, err)
	b, err = csv.Checksum(sum)
	assert.Nil(t, err)
	s := hex.EncodeToString(b[:])
	assert.Equal(t, sum, s)
}

func TestImport(t *testing.T) {
	var empty [32]byte

	s, path, err := csv.Import("", "")
	assert.NotNil(t, err)
	assert.Equal(t, "", path)
	assert.Equal(t, empty, s)

	// const file = "someimage.png"
	// const sum = "44dcc97a2b115c9fd51c95d6a3f2075f2f7c09067e34a33d9259cd22208bffba"
	// //abs := strings.Join([]string{path, file}, string(os.PathSeparator))
	// line := strings.Join([]string{sum, file}, ",")
	// chksum, err := csv.Checksum(line)
	// assert.Nil(t, err)

	// bucker1, err := mock.Bucket(1)
	// assert.Nil(t, err)
	// s, path, err = csv.Import(line, bucker1)
	// assert.Nil(t, err)
	// assert.NotEqual(t, "", path)
	// assert.Equal(t, chksum, s)

	//const helloWorld = "68656c6c6f20776f726c64"
	// 	helloWorld = "68656c6c6f20776f726c64"
	// )
	// path := "/home/me/downloads"
	// if runtime.GOOS == csv.WinOS {
	// 	path = "C:\\home\\me\\downloads"
	// }
	// abs := strings.Join([]string{path, file}, string(os.PathSeparator))
	// line := strings.Join([]string{sum, file}, ",")
	// bsum, err := csv.Checksum(sum)
	// if err != nil {
	// 	t.Error(err)
	// }
	//empty := [32]byte{}
	// type args struct {
	// 	line   string
	// 	bucket string
	// }
	// tests := []struct {
	// 	name     string
	// 	args     args
	// 	wantSum  [32]byte
	// 	wantPath string
	// 	wantErr  bool
	// }{
	// 	{"empty", args{"", ""}, empty, "", true},
	// 	{"invalid hash", args{helloWorld, ""}, empty, "", true},
	// 	{"invalid data", args{"invalid,csv data", path}, empty, "", true},
	// 	{"empty path", args{sum + ",", path}, bsum, path, false},
	// 	{"valid hash", args{line, path}, bsum, abs, false},
	// }
	// for _, tt := range tests {
	// 	t.Run(tt.name, func(t *testing.T) {
	// 		gotSum, gotPath, err := csv.Import(tt.args.line, tt.args.bucket)
	// 		if (err != nil) != tt.wantErr {
	// 			t.Errorf("Import() error = %v, wantErr %v", err, tt.wantErr)
	// 			return
	// 		}
	// 		if !reflect.DeepEqual(gotSum, tt.wantSum) {
	// 			t.Errorf("Import() gotSum = %v, want %v", gotSum, tt.wantSum)
	// 		}
	// 		if gotPath != tt.wantPath {
	// 			t.Errorf("Import() gotPath = %v, want %v", gotPath, tt.wantPath)
	// 		}
	// 	})
	// }
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
			got := csv.PathWindows(tt.path)
			assert.Equal(t, tt.want, got)
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
			got := csv.PathPosix(tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}
