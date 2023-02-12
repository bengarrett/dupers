// © Ben Garrett https://github.com/bengarrett/dupers
package archive_test

import (
	"log"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bengarrett/dupers/dupe"
	"github.com/bengarrett/dupers/dupe/internal/archive"
	"github.com/bengarrett/dupers/dupe/internal/parse"
	"github.com/bengarrett/dupers/internal/mock"
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
	if err := mock.TestOpen(); err != nil {
		t.Error(err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := archive.Extension(tt.find); got != tt.want {
				t.Errorf("extension() = %v, want %v", got, tt.want)
			}
		})
	}
	if err := mock.TestRemove(); err != nil {
		log.Fatal(err)
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
	type args struct {
		name parse.Bucket
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"empty", args{""}, true},
		{"bucket1", args{parse.Bucket(mock.Bucket1())}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := dupe.Config{
				Test: true,
			}
			db, err := mock.TestDB()
			if err != nil {
				t.Error(err)
			}
			defer db.Close()
			if err := c.WalkArchiver(nil, tt.args.name); (err != nil) != tt.wantErr {
				t.Errorf("Config.WalkArchiver() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigRead7Zip(t *testing.T) {
	c := dupe.Config{Test: true, Quiet: false, Debug: true}
	type args struct {
		bucket string
		name   string
	}
	tests := []struct {
		name string
		args args
	}{
		{"empty", args{}},
		{"7z", args{mock.Bucket1(), mock.SevenZip}},
	}
	if err := mock.TestOpen(); err != nil {
		t.Error(err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c.Read7Zip(nil, tt.args.bucket, tt.args.name)
		})
	}
	if err := mock.TestRemove(); err != nil {
		log.Fatal(err)
	}
}
