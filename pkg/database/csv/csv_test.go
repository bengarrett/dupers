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
	"github.com/stretchr/testify/assert"
)

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

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)

	_, _, err = csv.Import("", bucket1)
	assert.NotNil(t, err)

	_, _, err = csv.Import(mock.NoSuchFile, bucket1)
	assert.NotNil(t, err)

	sum, err := mock.ItemSum(0)
	assert.Nil(t, err)
	file, err := mock.Extension("txt")
	assert.Nil(t, err)
	mockCSVLine := strings.Join([]string{sum, file}, ",")

	_, path, err = csv.Import(mockCSVLine, bucket1)
	assert.Nil(t, err)
	assert.Contains(t, path, filepath.Base(file))
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
