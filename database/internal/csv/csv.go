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
	ErrChecksumLen  = errors.New("hexadecimal value is invalid")
	ErrFileNoDesc   = errors.New("no file descriptor")
	ErrImportFile   = errors.New("not a valid dupers export file")
	ErrImportPath   = errors.New("import item has an invalid file path")
	ErrImportSyntax = errors.New("import item has incorrect syntax")
)

const (
	Header = "sha256_sum,path#"
	WinOS  = "windows"

	BackSlash = "\u005C"
	FwdSlash  = "\u002F"
	UncPath   = BackSlash + BackSlash
)

type (
	// Filepath is the absolute path to a file used as a map key.
	Filepath string
)

// BucketName validates an export csv file header and returns the bucket name.
func BucketName(s string) string {
	const expected = 2
	ss := strings.Split(s, "#")
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
	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("%w: %s", err, file.Name())
	}
	b := make([]byte, l)
	_, err = io.ReadAtLeast(file, b, len(Header))
	if err != nil {
		return err
	}
	if !bytes.Equal(b, []byte(Header)) {
		return fmt.Errorf("%w, missing header: %s", ErrImportFile, Header)
	}
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("%w: %s", err, file.Name())
	}
	return nil
}

// Checksum returns a 64 character hexadecimal string as a bytes representation.
func Checksum(s string) ([32]byte, error) {
	const fixLength = 64
	empty, bs := [32]byte{}, [32]byte{}
	if len(s) != fixLength {
		return empty, fmt.Errorf("%w: value must contain exactly %d characters", ErrChecksumLen, fixLength)
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
	const drive = "C:"

	switch src {
	case "":
		return ""
	case "/":
		return drive
	}
	// source path is windows or unc
	if isDrive(src) {
		return src[0:2]
	}
	if isDrive(src[0:2]) {
		return src
	}
	if isUNC(src) {
		return src
	}
	// source path is posix
	src = strings.ReplaceAll(src, FwdSlash, BackSlash)
	if !isDrive(src[0:2]) {
		return drive + src
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
	ps := strings.SplitN(src, ":", subStr)
	drive, valid := ps[0], regexp.MustCompile(`^[a-z|A-Z]$`)
	if valid.MatchString(drive) {
		src = strings.ReplaceAll(ps[1], BackSlash, FwdSlash)
		if src == "" {
			src = FwdSlash
		}
	}
	return src
}

func isDrive(path string) bool {
	valid := regexp.MustCompile(`^[a-z|A-Z]:\\?$`)
	return valid.MatchString(path)
}

func isUNC(path string) bool {
	const uncLen = 2
	if len(path) < uncLen {
		return false
	}
	return (path[0:uncLen] == "\\\\")
}
