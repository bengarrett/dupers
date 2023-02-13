// © Ben Garrett https://github.com/bengarrett/dupers
package archive_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/bengarrett/dupers/dupe"
	"github.com/bengarrett/dupers/dupe/internal/archive"
	"github.com/bengarrett/dupers/dupe/internal/parse"
	"github.com/bengarrett/dupers/internal/mock"
)

const (
	bucket1 = "../../../test/bucket1/"
	file1   = "../../../test/bucket1/0vlLaUEvzAWP"
	file2   = "../../../test/bucket1/GwejJkMzs3yP"
	file7z  = "../../../test/randomfiles.7z"
)

func TestExtension(t *testing.T) {
	const xz = ".xz"
	tests := []struct {
		name string
		find string
		want string
	}{
		{"empty", "", ""},
		{"xz1", xz, archive.MimeXZ},
		{"xz2", archive.MimeXZ, xz},
		{"caps", strings.ToUpper(xz), archive.MimeXZ},
		{"no dot", "xz", xz},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := archive.Extension(tt.find); got != tt.want {
				t.Errorf("extension() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadMIME(t *testing.T) {
	dir, err := filepath.Abs("../../../test")
	if err != nil {
		t.Error(err)
		return
	}
	tests := []struct {
		name     string
		filename string
		wantMime string
		wantErr  bool
	}{
		{"empty", "", "", true},
		{"text file", filepath.Join(dir, "randomfiles.txt"), "", true},
		{"7z", filepath.Join(dir, "randomfiles.7z"), archive.Mime7z, false},
		{"xz", filepath.Join(dir, "randomfiles.tar.xz"), archive.MimeXZ, true},
		{"zip", filepath.Join(dir, "randomfiles.zip"), archive.MimeZip, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMime, err := archive.ReadMIME(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadMIME() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotMime != tt.wantMime {
				t.Errorf("ReadMIME() gotMime = %v, want %v", gotMime, tt.wantMime)
			}
		})
	}
}

func TestMIME(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantMime string
	}{
		{"empty", "", ""},
		{"text", "file.txt", ""},
		{"zip", "file.zip", archive.MimeZip},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMime := archive.MIME(tt.filename)
			if gotMime != tt.wantMime {
				t.Errorf("IsExtension() gotMime = %v, want %v", gotMime, tt.wantMime)
			}
		})
	}
}

func TestConfig_WalkArchiver(t *testing.T) {
	db, err := mock.Database()
	if err != nil {
		t.Error(err)
	}
	defer db.Close()
	type args struct {
		name parse.Bucket
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"empty", args{""}, true},
		{"non-exist", args{"this-directory-does-not-exist"}, true},
		{"file", args{parse.Bucket(mock.Item(1))}, false},
		{"bucket1", args{parse.Bucket(mock.Bucket1())}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := dupe.Config{Test: true}
			if err := c.WalkArchiver(db, tt.args.name); (err != nil) != tt.wantErr {
				t.Errorf("Config.WalkArchiver() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigRead7Zip(t *testing.T) {
	c := dupe.Config{Test: true, Quiet: false, Debug: true}
	type args struct {
		bucket parse.Bucket
		name   string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"empty", args{}, true},
		{"file", args{file2, ""}, true},
		{"file+bucket", args{file2, bucket1}, true},
		{"dir", args{bucket1, ""}, true},
		{"7Z no bucket", args{"", file7z}, true},
		{"7Z", args{parse.Bucket(mock.Bucket1()), file7z}, false},
	}
	db, err := mock.Database()
	if err != nil {
		t.Error(err)
	}
	defer db.Close()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Read7Zip(db, tt.args.bucket, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("Read7Zip error = %v, want %v: %v", (err != nil), tt.wantErr, err)
			}
		})
	}
}
