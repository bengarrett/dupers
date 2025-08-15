// © Ben Garrett https://github.com/bengarrett/dupers
package csv

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var (
	ErrHexDec       = errors.New("hexadecimal value is invalid")
	ErrFileNoDesc   = errors.New("no file descriptor")
	ErrImportFile   = errors.New("not a valid dupers export file")
	ErrImportPath   = errors.New("import item has an invalid file path")
	ErrImportSyntax = errors.New("import item has incorrect syntax")
)

const (
	Header    = "sha256_sum,path#"    // Header that is inserted into exported CSV files.
	WinOS     = "windows"             // WinOS is the Windows operating system.s
	BackSlash = "\u005C"              // BackSlash Unicode representation.
	FwdSlash  = "\u002F"              // FwdSlash is a forward slash Unicode representation.
	UncPath   = BackSlash + BackSlash // UncPath is the  Universal Naming Convention path.
)

type Filepath string // Filepath is the absolute path to a file used as a map key.

// Bucket validates the header of a csv file and returns the embedded bucket name.
func Bucket(header string) string {
	const expected = 2
	ss := strings.Split(header, "#")
	if len(ss) != expected {
		return ""
	}
	path := ss[1]
	if runtime.GOOS == WinOS {
		path = PathWindows(path)
	} else {
		path = PathPosix(path)
	}
	if filepath.IsAbs(path) {
		return path
	}
	return ""
}

// Checker reads the first line in the export csv file and returns nil if it uses the expected syntax.
func Checker(file *os.File) error {
	if file == nil {
		return ErrFileNoDesc
	}
	l := len(Header)
	b := make([]byte, l)
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("%w: %s", err, file.Name())
	}
	if _, err := io.ReadAtLeast(file, b, len(Header)); err != nil {
		return err
	}
	if !bytes.Equal(b, []byte(Header)) {
		return fmt.Errorf("%w, missing header: %s", ErrImportFile, Header)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("%w: %s", err, file.Name())
	}
	return nil
}

// Checksum returns a 64 character hexadecimal string as a bytes representation.
func Checksum(s string) ([32]byte, error) {
	const fixLength = 64
	empty, bs := [32]byte{}, [32]byte{}
	if len(s) != fixLength {
		return empty, fmt.Errorf("%w: value must contain exactly %d characters", ErrHexDec, fixLength)
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return empty, err
	}
	copy(bs[:], b)
	return bs, nil
}

// Import reads, validates and returns a line of data from an export csv file.
func Import(line, bucket string) (sum [32]byte, path string, err error) {
	const expected = 2
	empty := [32]byte{}
	ss := strings.Split(line, ",")
	if len(ss) != expected {
		return empty, "", ErrImportSyntax
	}
	name := filepath.Join(bucket, ss[1])
	if !filepath.IsAbs(name) {
		return empty, "", ErrImportPath
	}
	sum, err = Checksum(ss[0])
	if err != nil {
		return empty, "", err
	}
	return sum, name, nil
}

// PathWindows returns a Windows or UNC usable path from the source path.
func PathWindows(src string) string {
	const path = "C:"

	switch src {
	case "":
		return ""
	case "/":
		return path
	}
	// source path is windows or unc
	if drive(src) {
		return src[0:2]
	}
	if drive(src[0:2]) {
		return src
	}
	if unc(src) {
		return src
	}
	// source path is posix
	src = strings.ReplaceAll(src, FwdSlash, BackSlash)
	if !drive(src[0:2]) {
		return path + src
	}
	return src
}

// PathPosix returns POSIX path from the source path.
func PathPosix(src string) string {
	const driveLen, subStr = 2, 2
	if len(src) < driveLen {
		return src
	}
	if src[0:driveLen] == UncPath {
		return fmt.Sprintf("%s%s", FwdSlash,
			strings.ReplaceAll(src[driveLen:], BackSlash, FwdSlash))
	}
	drive, after, _ := strings.Cut(src, ":")
	valid := regexp.MustCompile(`^[a-z|A-Z]$`)
	if valid.MatchString(drive) {
		src = strings.ReplaceAll(after, BackSlash, FwdSlash)
		if src == "" {
			src = FwdSlash
		}
	}
	return src
}

func drive(path string) bool {
	valid := regexp.MustCompile(`^[a-z|A-Z]:\\?$`)
	return valid.MatchString(path)
}

func unc(path string) bool {
	const uncLen = 2
	if len(path) < uncLen {
		return false
	}
	return (path[0:uncLen] == "\\\\")
}
