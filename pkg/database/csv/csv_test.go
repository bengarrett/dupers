// © Ben Garrett https://github.com/bengarrett/dupers
package csv_test

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bengarrett/dupers/internal/mock"
	"github.com/bengarrett/dupers/pkg/database/csv"
	"github.com/nalgeon/be"
)

func TestBucket(t *testing.T) {
	s := csv.Bucket("")
	be.Equal(t, s, "")
	const badHeader = "this is an invalid csv file header"
	s = csv.Bucket(badHeader)
	be.Equal(t, s, "")
	s = csv.Bucket(csv.Header)
	be.Equal(t, s, "")
	bucket1, err := mock.Bucket(1)
	be.Err(t, err, nil)
	s = csv.Bucket(csv.Header + bucket1)
	be.Equal(t, s, bucket1)
}

func TestChecker(t *testing.T) {
	err := csv.Checker(nil)
	be.Err(t, err)
	item1, err := mock.Item(1)
	be.Err(t, err, nil)
	binary, err := os.Open(item1)
	be.Err(t, err, nil)
	defer binary.Close()
	err = csv.Checker(binary)
	be.Err(t, err)
	text, err := os.Open(mock.CSV())
	be.Err(t, err, nil)
	defer binary.Close()
	err = csv.Checker(text)
	be.Err(t, err, nil)
}

func TestChecksum(t *testing.T) {
	var empty [32]byte
	zeros := strings.Repeat("0", 32)
	b, err := csv.Checksum("")
	be.Err(t, err)
	be.Equal(t, b, empty)
	b, err = csv.Checksum(zeros)
	be.Err(t, err)
	be.Equal(t, b, empty)
	sum, err := mock.ItemSum(1)
	be.Err(t, err, nil)
	b, err = csv.Checksum(sum)
	be.Err(t, err, nil)
	s := hex.EncodeToString(b[:])
	be.Equal(t, s, sum)
}

func TestImport(t *testing.T) {
	var empty [32]byte
	s, path, err := csv.Import("", "")
	be.Err(t, err)
	be.Equal(t, path, "")
	be.Equal(t, s, empty)
	bucket1, err := mock.Bucket(1)
	be.Err(t, err, nil)
	_, _, err = csv.Import("", bucket1)
	be.Err(t, err)
	_, _, err = csv.Import(mock.NoSuchFile, bucket1)
	be.Err(t, err)
	sum, err := mock.ItemSum(0)
	be.Err(t, err, nil)
	file, err := mock.Extension("txt")
	be.Err(t, err, nil)
	mockCSVLine := strings.Join([]string{sum, file}, ",")
	_, path, err = csv.Import(mockCSVLine, bucket1)
	be.Err(t, err, nil)
	ok := strings.Contains(path, filepath.Base(file))
	be.True(t, ok)
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
			be.Equal(t, got, tt.want)
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
			be.Equal(t, got, tt.want)
		})
	}
}
